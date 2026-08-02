package dto

// PresignedURLRequest is the request body for POST /api/upload/presigned-url.
// The frontend sends the file metadata before uploading directly to R2.
type PresignedURLRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	ContentType string `json:"content_type" validate:"required,oneof=image/jpeg image/png image/webp"`
	Size        int64  `json:"size" validate:"required,min=1,max=5242880"`
}

// PresignedURLResponse is returned after a successful presigned URL request.
// The frontend PUTs the file to UploadURL, then submits the PublicURL as
// cover_image_url in the event creation payload.
type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
}
