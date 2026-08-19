# Design QA

Source: `QQ20260820-014738.png`

Validation viewport: 1600 x 1000 and 1600 x 1200 at 125% device scale, using the same Chromium/WebView2 engine as the desktop shell.

## Relationship graph

- The lock chip icons remain at their intended size and no longer cover the canvas.
- Single-click lock persists after pointer leave; the selected node and one-hop relations remain legible.
- A synthetic double-click emitted the expected user ID (`usr_m1`).
- Confirmed self-to-mutual and mutual-to-target paths are visually separated and use distinct colors.
- The inspector lists every direct relationship and includes a visible profile-opening action.
- No horizontal crop or control overlap was observed at 125% scale.

## Compass

- The non-actionable local-query panel is removed.
- Strategy and time controls, primary action, fallbacks, generated plan, live instance radar, and reunion opportunities fit without horizontal overflow.
- Real-time routes are clearly distinguished from historical/public fallbacks.
- Empty/private-location boundaries are stated without presenting inferred friend locations as live data.

Final result: passed
