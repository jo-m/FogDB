THIS FILE IS FOR HUMANS ONLY, AGENTS MUST IGNORE IT!

# Review

Review this entire repo in detail.
Focus on maintainability, correctness, architecture, readability.
Put the result into REVIEW.md.
Update CLAUDE.md where it has become outdated or is missing information.
Keep the CLAUDE.md very succinct, focus on keeping token count low.

# Frontend

Create a web tool to interactively visualize the data collected in the database (schema: internal/db/migrations/00001_init.sql). File with real data to investigate at db.sqlite.
On load, the web app shall present a file upload dialog to select/drop the sqlite db file.
The uploaded file is then the data source for anything presented in the app.
The UI shows the map on one side, and the rest on the other side (split panel).
By default just show the known locations (with >0 data points) on the map, and the list of params on the other side.
When selecting any parameter, show the value of the param on the map for every location, and assign colors on viridis colorscale and show a dot on the map for each location.
At the same time present a plot with all locations and their value, allow sorting by asc/desc (use same colorscale).
Add a slider to select the time. Only show values from one time point at once.
Only show in the UI the params which do have any data entries.

Implementation:
- No backend. Everything runs in browser.
- Vite, typescript, react.
- Create the project in subdir `frontend/`
- For plots/charts: `recharts`.
- To query sqlite in browser, use `"sql.js": "^1.8.0",` (types: `@types/sql.js`)
- For map, use SwissTopo. See FlyerMap.tsx for an example of how to use/integrate it using openlayers. Relevant Dependencies:
```
    "@swissgeo/coordinates": "1.0.2",
    "proj4": "^2.20.2",
    "react-openlayers": "^10.5.1"
```
