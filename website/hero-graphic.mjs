// The hero graphic of the home page: one registry (the mestiere mark) on the left, threads
// fanning out to the repositories of a team on the right, and a pulse running along each thread:
// a skill being installed. Pure SVG built at build time; motion is SMIL (no script, allowed by the
// CSP) and is hidden by CSS under prefers-reduced-motion. Colors follow currentColor.

const W = 640;
const H = 460;
const SOURCE = { x: 110, y: H / 2, r: 30 };
const REPOS = ['web', 'api', 'mobile', 'billing', 'search', 'auth', 'infra', 'data', 'docs'];
const DURATION = 3.2; // seconds a pulse takes to reach a repository
const STAGGER = 0.41; // seconds between two pulses
const START = 0.9; // seconds before the first pulse: the graphic fades in first, still

const round = (n) => Math.round(n * 10) / 10;

// Mark path from docs/assets/mestiere-mark-black.svg (viewBox 87, no margin)
const MARK = 'M43.6781 25.9985C41.4135 25.9985 38.9332 27.2926 38.5019 29.6651C37.9627 32.5767 39.1489 36.0275 39.3645 39.047C39.5802 42.7136 39.6881 46.3801 39.7959 49.9388C39.7959 49.5074 38.6097 48.5369 38.2862 48.2133C37.747 47.5663 37.2078 46.9193 36.6686 46.3801C35.5902 45.086 34.5118 43.8998 33.4334 42.6057C32.355 41.3117 31.2767 40.0176 30.1983 38.8314C29.2277 37.753 28.4728 36.3511 27.2866 35.3805C25.1298 33.5473 21.5712 34.7335 20.1692 36.8903C18.6595 39.3706 19.3065 42.6057 21.7868 44.1155C26.5317 47.4585 30.7374 51.4485 34.9432 55.3307C34.6196 55.0072 33.002 55.2229 32.5707 55.2229C31.708 55.2229 30.9531 55.2229 30.0904 55.115C28.365 55.115 26.6396 54.8993 25.022 54.7915C21.679 54.5758 18.336 54.2523 14.8851 53.9288C13.3754 53.821 11.9735 53.9288 11.0029 55.0072C10.0324 56.1934 9.81672 57.9188 10.1402 59.4286C11.4343 65.8989 27.0709 61.5854 43.6781 61.2618C60.2853 61.5854 76.0297 65.8989 77.216 59.4286C77.5395 57.9188 77.3238 56.1934 76.3533 55.0072C75.3827 53.821 73.873 53.821 72.4711 53.9288C69.1281 54.2523 66.2164 54.5758 62.8734 54.7915C59.7461 55.115 52.8444 55.115 52.413 55.3307C52.8444 54.6837 60.9323 47.4585 65.5694 44.1155C67.834 42.2822 68.6967 39.3706 67.187 36.8903C65.7851 34.6256 62.2264 33.5473 60.0696 35.3805C58.9912 36.3511 58.1285 37.753 57.158 38.8314C56.0796 40.1254 55.0012 41.4195 53.9228 42.6057C52.8444 43.8998 51.766 45.086 50.6876 46.3801C50.1484 47.0271 49.6093 47.6741 49.0701 48.2133C48.7465 48.5369 47.5603 49.5074 47.5603 49.9388C47.6681 46.2722 47.776 42.6057 47.9916 39.047C48.2073 36.0275 49.3936 32.6845 48.8544 29.6651C48.423 27.2926 45.8349 26.1064 43.6781 25.9985Z';

export function heroGraphic(title = 'One skills registry feeding every repository of a team') {
  const top = 36;
  const step = (H - 2 * top) / (REPOS.length - 1);
  const repos = REPOS.map((name, i) => {
    const y = top + i * step;
    const t = (y - H / 2) / (H / 2 - top); // -1..1
    const x = 470 - 56 * t * t; // repositories sit on a gentle arc
    const start = { x: SOURCE.x + SOURCE.r + 6, y: SOURCE.y };
    const d = `M${start.x} ${start.y} C${start.x + 150} ${start.y} ${round(x - 170)} ${round(y)} ${round(x - 9)} ${round(y)}`;
    return { name, x: round(x), y: round(y), d, begin: round(START + (i * STAGGER * 2 % (REPOS.length * STAGGER))) };
  });

  const threads = repos.map((r) => `<path class="loom-thread" d="${r.d}"/>`).join('');
  const pulses = repos.map((r) => `<circle class="loom-pulse" r="3">
        <animateMotion dur="${DURATION}s" begin="${r.begin}s" repeatCount="indefinite" path="${r.d}" calcMode="spline" keyPoints="0;1" keyTimes="0;1" keySplines="0.45 0 0.2 1"/>
        <animate attributeName="opacity" dur="${DURATION}s" begin="${r.begin}s" repeatCount="indefinite" values="0;1;1;0" keyTimes="0;0.08;0.9;1"/>
      </circle>`).join('');
  const nodes = repos.map((r) => `<g class="loom-repo">
        <circle class="loom-node" cx="${r.x}" cy="${r.y}" r="4.5">
          <animate attributeName="r" dur="${DURATION}s" begin="${r.begin}s" repeatCount="indefinite" values="4.5;4.5;8;4.5" keyTimes="0;0.9;0.95;1"/>
        </circle>
        <circle class="loom-ring" cx="${r.x}" cy="${r.y}" r="4.5">
          <animate attributeName="r" dur="${DURATION}s" begin="${r.begin}s" repeatCount="indefinite" values="4.5;4.5;18" keyTimes="0;0.92;1"/>
          <animate attributeName="opacity" dur="${DURATION}s" begin="${r.begin}s" repeatCount="indefinite" values="0;0;0.6;0" keyTimes="0;0.92;0.93;1"/>
        </circle>
        <text class="loom-label" x="${r.x + 16}" y="${r.y}">${r.name}/</text>
      </g>`).join('');

  const scale = 0.62;
  const mark = `<path class="loom-mark" transform="translate(${round(SOURCE.x - 43.6 * scale)} ${round(SOURCE.y - 44.5 * scale)}) scale(${scale})" d="${MARK}"/>`;

  return `<svg class="loom" viewBox="0 0 ${W} ${H}" role="img" aria-labelledby="loom-title">
        <title id="loom-title">${title}</title>
        <g class="loom-threads">${threads}</g>
        <g class="loom-pulses">${pulses}</g>
        <circle class="loom-halo" cx="${SOURCE.x}" cy="${SOURCE.y}" r="${SOURCE.r}">
          <animate attributeName="r" dur="${DURATION / 2}s" begin="${START}s" repeatCount="indefinite" values="${SOURCE.r};${SOURCE.r + 16}"/>
          <animate attributeName="opacity" dur="${DURATION / 2}s" begin="${START}s" repeatCount="indefinite" values="0.25;0"/>
        </circle>
        <circle class="loom-source" cx="${SOURCE.x}" cy="${SOURCE.y}" r="${SOURCE.r}"/>
        ${mark}
        <text class="loom-caption" x="${SOURCE.x}" y="${SOURCE.y + SOURCE.r + 26}">registry</text>
        ${nodes}
      </svg>`;
}

// The 404 graphic: the same registry and threads, but the middle thread is snapped. Its pulse
// runs to the break and fades out, the loose end sways, and the repository it should reach,
// "404/", is only a dashed outline.
export function notFoundGraphic(title, idSuffix = '') {
  const W2 = 560;
  const H2 = 420;
  const src = { x: 96, y: H2 / 2, r: 30 };
  const names = ['web', 'api', '404', 'docs', 'infra'];
  const start = { x: src.x + src.r + 6, y: src.y };
  const breakAt = { x: 300, y: src.y + 3 };
  const titleId = `loom-404-title${idSuffix}`;
  const rows = names.map((name, i) => {
    const y = 70 + i * 70;
    const t = (y - src.y) / 140;
    const x = round(450 - 34 * t * t);
    const d = `M${start.x} ${start.y} C${start.x + 120} ${start.y} ${round(x - 150)} ${y} ${round(x - 9)} ${y}`;
    return { name, x, y, d, begin: round(START + i * 0.7), missing: name === '404' };
  });

  const pulse = (d, begin, values = '0;1;1;0', keyTimes = '0;0.08;0.9;1') => `<circle class="loom-pulse" r="3">
        <animateMotion dur="${DURATION}s" begin="${begin}s" repeatCount="indefinite" path="${d}" calcMode="spline" keyPoints="0;1" keyTimes="0;1" keySplines="0.45 0 0.2 1"/>
        <animate attributeName="opacity" dur="${DURATION}s" begin="${begin}s" repeatCount="indefinite" values="${values}" keyTimes="${keyTimes}"/>
      </circle>`;

  const parts = rows.map((r) => {
    if (r.missing) {
      const head = `M${start.x} ${start.y} C${start.x + 90} ${start.y} ${breakAt.x - 70} ${breakAt.y - 2} ${breakAt.x} ${breakAt.y}`;
      return `<path class="loom-thread" d="${head}"/>
        <g class="loom-loose">
          <animateTransform attributeName="transform" type="rotate" dur="3.6s" repeatCount="indefinite" calcMode="spline" keyTimes="0;0.5;1" keySplines="0.45 0 0.55 1;0.45 0 0.55 1" values="-7 ${breakAt.x + 12} ${breakAt.y + 4};7 ${breakAt.x + 12} ${breakAt.y + 4};-7 ${breakAt.x + 12} ${breakAt.y + 4}"/>
          <path class="loom-thread loom-thread-loose" d="M${breakAt.x + 12} ${breakAt.y + 4} C${breakAt.x + 26} ${breakAt.y + 10} ${breakAt.x + 34} ${breakAt.y + 26} ${breakAt.x + 36} ${breakAt.y + 46}"/>
        </g>
        ${pulse(head, r.begin, '0;1;1;0', '0;0.1;0.75;1')}
        <circle class="loom-missing" cx="${r.x}" cy="${r.y}" r="6"/>
        <text class="loom-label loom-label-missing" x="${r.x + 16}" y="${r.y}">${r.name}/</text>`;
    }
    return `<path class="loom-thread" d="${r.d}"/>
        ${pulse(r.d, r.begin)}
        <circle class="loom-node" cx="${r.x}" cy="${r.y}" r="4.5"/>
        <text class="loom-label" x="${r.x + 16}" y="${r.y}">${r.name}/</text>`;
  }).join('\n        ');

  const scale = 0.62;
  const mark = `<path class="loom-mark" transform="translate(${round(src.x - 43.6 * scale)} ${round(src.y - 44.5 * scale)}) scale(${scale})" d="${MARK}"/>`;
  return `<svg class="loom loom-404" viewBox="0 0 ${W2} ${H2}" role="img" aria-labelledby="${titleId}">
        <title id="${titleId}">${title}</title>
        ${parts}
        <circle class="loom-source" cx="${src.x}" cy="${src.y}" r="${src.r}"/>
        ${mark}
      </svg>`;
}
