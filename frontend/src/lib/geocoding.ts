export interface PlaceResult {
  venueName: string;
  locationAddress: string;
  lat: number;
  lng: number;
}

interface PhotonProperties {
  name?: string;
  street?: string;
  housenumber?: string;
  city?: string;
  state?: string;
  country?: string;
  postcode?: string;
  district?: string;
}

interface PhotonFeature {
  properties: PhotonProperties;
  geometry: { coordinates: [number, number] };
}

interface PhotonResponse {
  features: PhotonFeature[];
}

function formatPhotonAddress(props: PhotonProperties): string {
  const line1 = props.housenumber
    ? [props.housenumber, props.street].filter(Boolean).join(" ")
    : props.street;
  const parts = [line1, props.district, props.city, props.state, props.postcode, props.country].filter(
    Boolean,
  );
  return parts.join(", ");
}

function venueFromPhoton(props: PhotonProperties, address: string): string {
  if (props.name && props.name !== props.street) return props.name;
  if (props.street) return props.street;
  return address.split(",")[0]?.trim() || "Venue";
}

function photonToPlace(feature: PhotonFeature): PlaceResult {
  const address = formatPhotonAddress(feature.properties);
  const [lng, lat] = feature.geometry.coordinates;
  return {
    venueName: venueFromPhoton(feature.properties, address),
    locationAddress: address,
    lat,
    lng,
  };
}

export async function searchPlaces(query: string, signal?: AbortSignal): Promise<PlaceResult[]> {
  const trimmed = query.trim();
  if (trimmed.length < 2) return [];

  const mapboxToken = import.meta.env.VITE_MAPBOX_TOKEN as string | undefined;
  if (mapboxToken) {
    return searchMapbox(trimmed, mapboxToken, signal);
  }
  return searchPhoton(trimmed, signal);
}

async function searchPhoton(query: string, signal?: AbortSignal): Promise<PlaceResult[]> {
  const url = new URL("https://photon.komoot.io/api/");
  url.searchParams.set("q", query);
  url.searchParams.set("limit", "6");
  url.searchParams.set("lang", "en");

  const res = await fetch(url, { signal });
  if (!res.ok) throw new Error("Address search failed");
  const data = (await res.json()) as PhotonResponse;
  return data.features.map(photonToPlace);
}

interface MapboxFeature {
  place_name: string;
  text: string;
  center: [number, number];
}

interface MapboxResponse {
  features: MapboxFeature[];
}

async function searchMapbox(
  query: string,
  token: string,
  signal?: AbortSignal,
): Promise<PlaceResult[]> {
  const url = new URL(
    `https://api.mapbox.com/geocoding/v5/mapbox.places/${encodeURIComponent(query)}.json`,
  );
  url.searchParams.set("access_token", token);
  url.searchParams.set("autocomplete", "true");
  url.searchParams.set("limit", "6");
  url.searchParams.set("types", "address,poi,place");

  const res = await fetch(url, { signal });
  if (!res.ok) throw new Error("Address search failed");
  const data = (await res.json()) as MapboxResponse;
  return data.features.map((f) => ({
    venueName: f.text,
    locationAddress: f.place_name,
    lat: f.center[1],
    lng: f.center[0],
  }));
}