# Brand assets

| File in [`assets/`](assets) | Use |
|---|---|
| `mestiere-wordmark-black.svg`, `mestiere-wordmark-white.svg` | README header, light and dark |
| `mestiere-mark-black.svg`, `mestiere-mark-white.svg` | The mark alone, light and dark (website hero graphic) |
| `mestiere-hero.png`, `mestiere-hero-dark.png` | README banner, light and dark |
| `mestiere-social.png` | GitHub social preview, uploaded in the repository settings |

Wordmark and mark are cropped to the artwork, with no margin: add the space where you place them.

Colors: black `#000000`, white `#FFFFFF`. Typefaces: Parmigiano Text L for display text, a system monospace for paths.

## Rebuilding the banner and social preview

Sources are in [`assets/src/`](assets/src). They reference Parmigiano Text L by name, so it must be installed or visible to fontconfig when rendering. The font isn't in the repository because of its license: keep the OTFs in `docs/assets/fonts/`, which git ignores. The website's woff2 files in `website/static/fonts/` are converted from them.

```sh
cd docs/assets
rsvg-convert src/hero.svg -o mestiere-hero.png
sed -e 's/#fff/#XXX/g' -e 's/#000/#fff/g' -e 's/#XXX/#000/g' src/hero.svg | rsvg-convert -o mestiere-hero-dark.png
rsvg-convert src/social.svg -o mestiere-social.png
```
