# FogDB Viewer

Browser-only visualizer for a FogDB SQLite archive. On load it prompts for a
database file; everything runs client-side with `sql.js` (WebAssembly SQLite).

## Run

```sh
npm install
npm run dev
```

Then open the printed URL and drop a `db.sqlite` file into the dialog.

## Scripts

- `npm run dev` - start the Vite dev server.
- `npm run build` - typecheck and produce a production build in `dist/`.
- `npm run test` - run unit tests (vitest).
- `npm run typecheck` - typecheck only.

## Notes

- The map uses SwissTopo tiles (WMTS, LV95/EPSG:2056) via OpenLayers.
- The chart is rendered with recharts; map and chart share a viridis scale.
- Only parameters with at least one forecast row are shown.
