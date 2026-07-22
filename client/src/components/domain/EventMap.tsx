import { MapContainer, Marker, Popup, TileLayer } from "react-leaflet";
import L from "leaflet";

const customIcon = L.divIcon({
  className: "",
  html: `<div class="grid size-8 -translate-x-1/2 -translate-y-full place-items-center">
    <span class="grid size-7 place-items-center rounded-full bg-primary text-primary-foreground shadow-lg ring-2 ring-background">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="size-4"><path d="M20 10c0 4.9-8 12-8 12s-8-7.1-8-12a8 8 0 0 1 16 0Z"/><circle cx="12" cy="10" r="3"/></svg>
    </span>
  </div>`,
  iconSize: [32, 32],
  iconAnchor: [16, 32],
});

export function EventMap({
  lat,
  lng,
  label,
}: {
  lat: number;
  lng: number;
  label?: string;
}) {
  return (
    <MapContainer
      center={[lat, lng]}
      zoom={15}
      scrollWheelZoom={false}
      className="h-64 w-full rounded-xl border"
      aria-label="Event location map"
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />
      <Marker position={[lat, lng]} icon={customIcon}>
        {label && <Popup>{label}</Popup>}
      </Marker>
    </MapContainer>
  );
}