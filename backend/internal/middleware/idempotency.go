// Package middleware provides an idempotency-key middleware backed by Redis,
// built for the Echo framework.
//
// Flow:
//  1. Client sends header `Idempotency-Key: <uuid>` + JSON body.
//  2. We hash the body, run a Lua script that atomically:
//     - if key doesn't exist -> claim it (status=in_progress), proceed
//     - if key exists + status=in_progress -> caller gets 409
//     - if key exists + status=done + same payload hash -> replay stored response
//     - if key exists + payload hash differs -> 409 (key reused, different payload)
//  3. Handler runs normally, response is captured via a ResponseWriter wrapper.
//  4. On completion (success OR failure), we overwrite the key with the final
//     response and a 5-min TTL. We do NOT rely on the in_progress TTL expiring
//     in the happy path — that TTL only exists to recover from crashed requests.
package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

const (
	idemInProgressTTL = 60 * time.Second
	idemDoneTTL       = 5 * time.Minute
	idemHeaderName    = "Idempotency-Key"
)

// claimScript atomically inspects/claims an idempotency key.
// KEYS[1] = redis key (idem:<id>)
// ARGV[1] = incoming payload hash
// ARGV[2] = in_progress TTL (seconds)
//
// Returns a table: {status, storedPayloadHash, storedResponseJSON}
// status is one of: "new", "in_progress", "done", "mismatch"
var claimScript = redis.NewScript(`
local existing = redis.call('GET', KEYS[1])
if not existing then
  local doc = cjson.encode({status = "in_progress", payload_hash = ARGV[1]})
  redis.call('SET', KEYS[1], doc, 'EX', ARGV[2])
  return {"new", "", ""}
end

local decoded = cjson.decode(existing)

if decoded.payload_hash ~= ARGV[1] then
  return {"mismatch", decoded.payload_hash, ""}
end

if decoded.status == "in_progress" then
  return {"in_progress", decoded.payload_hash, ""}
end

-- status == "done"
return {"done", decoded.payload_hash, cjson.encode(decoded.response)}
`)

type idemStoredResponse struct {
	StatusCode int             `json:"status_code"`
	Body       json.RawMessage `json:"body"`
	ContentTyp string          `json:"content_type"`
}

// idemResponseRecorder captures what the downstream handler writes so we can
// both send it to the real client and persist it to Redis afterward.
type idemResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (r *idemResponseRecorder) WriteHeader(code int) {
	if r.statusCode == http.StatusOK && code != 0 {
		r.statusCode = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *idemResponseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// Idempotency returns Echo middleware that enforces idempotency via Redis.
// When rdb is nil (no Redis configured), the middleware is a pass-through.
func Idempotency(rdb *redis.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			key := req.Header.Get(idemHeaderName)
			if key == "" || rdb == nil {
				return next(c)
			}

			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "failed to read body")
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			payloadHash := hashSHA256(bodyBytes)
			redisKey := "idem:" + key

			out, err := claimScript.Run(
				req.Context(),
				rdb,
				[]string{redisKey},
				payloadHash,
				int(idemInProgressTTL.Seconds()),
			).Slice()
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "idempotency check failed")
			}

			status, _ := out[0].(string)

			switch status {
			case "mismatch":
				return echo.NewHTTPError(http.StatusConflict, "idempotency key reused with a different payload")

			case "in_progress":
				c.Response().Header().Set("Retry-After", "2")
				return echo.NewHTTPError(http.StatusConflict, "request with this idempotency key is already in progress")

			case "done":
				storedJSON, _ := out[2].(string)
				var sr idemStoredResponse
				if jsonErr := json.Unmarshal([]byte(storedJSON), &sr); jsonErr != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "failed to replay stored response")
				}
				if sr.ContentTyp != "" {
					c.Response().Header().Set("Content-Type", sr.ContentTyp)
				}
				c.Response().Header().Set("Idempotent-Replayed", "true")
				return c.Blob(sr.StatusCode, sr.ContentTyp, sr.Body)

			case "new":
				// fall through to actually run the handler
			}

			res := c.Response()
			rec := &idemResponseRecorder{ResponseWriter: res.Writer, statusCode: http.StatusOK}
			res.Writer = rec

			handlerErr := next(c)

			finalizeIdempotency(req.Context(), rdb, redisKey, payloadHash, rec)

			return handlerErr
		}
	}
}

func finalizeIdempotency(ctx context.Context, rdb *redis.Client, redisKey, payloadHash string, rec *idemResponseRecorder) {
	sr := idemStoredResponse{
		StatusCode: rec.statusCode,
		Body:       rec.body.Bytes(),
		ContentTyp: rec.Header().Get("Content-Type"),
	}

	doc := map[string]any{
		"status":       "done",
		"payload_hash": payloadHash,
		"response":     sr,
	}

	encoded, err := json.Marshal(doc)
	if err != nil {
		return
	}

	rdb.Set(ctx, redisKey, encoded, idemDoneTTL)
}

func hashSHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
