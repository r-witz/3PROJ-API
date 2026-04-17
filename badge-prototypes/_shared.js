// Shared data for badge shape prototypes.
// Palette is aligned to the Duskforge FRONT design language:
//   surface #0f0f0f/#1a1a1a/#262626, text #f9fafb,
//   primary #6366f1 (indigo), secondary #8b5cf6 (violet), accent #f59e0b (amber).
//
// This iteration focuses on SHAPES only — no glyph icons inside the frame.
// A subtle placeholder ring marks where an icon would eventually sit.

const BADGES = [
  { code: "first_review",         name: "First Impressions",  tier: "bronze",   cat: "reviewing"  },
  { code: "prolific_reviewer",    name: "Prolific Reviewer",  tier: "silver",   cat: "reviewing"  },
  { code: "century_club_reviews", name: "Century Club",       tier: "gold",     cat: "reviewing"  },
  { code: "masterpiece_hunter",   name: "Masterpiece Hunter", tier: "silver",   cat: "reviewing"  },
  { code: "first_watch",          name: "Lights, Camera",     tier: "bronze",   cat: "watching"   },
  { code: "cinephile",            name: "Cinephile",          tier: "silver",   cat: "watching"   },
  { code: "movie_buff",           name: "Movie Buff",         tier: "gold",     cat: "watching"   },
  { code: "marathon_runner",      name: "Marathon Runner",    tier: "silver",   cat: "watching"   },
  { code: "screen_sage",          name: "Screen Sage",        tier: "platinum", cat: "watching"   },
  { code: "first_fan",            name: "First Fan",          tier: "bronze",   cat: "social"     },
  { code: "crowd_pleaser",        name: "Crowd Pleaser",      tier: "silver",   cat: "social"     },
  { code: "first_follower",       name: "Making Friends",     tier: "bronze",   cat: "social"     },
  { code: "community_voice",      name: "Community Voice",    tier: "gold",     cat: "social"     },
  { code: "conversationalist",    name: "Conversationalist",  tier: "silver",   cat: "social"     },
  { code: "curator",              name: "Curator",            tier: "silver",   cat: "collecting" }
];

// FRONT-aligned tiers: bronze=warm amber-copper, silver=slate, gold=amber (brand),
// platinum=violet (brand rating color, reserved for the rarest tier).
const TIER_COLORS = {
  bronze: {
    base:  "#c47e3a",
    light: "#e89a54",
    dark:  "#7a4a1a",
    tint:  "rgba(245, 158, 11, 0.16)"   // amber glow
  },
  silver: {
    base:  "#94a3b8",
    light: "#cbd5e1",
    dark:  "#475569",
    tint:  "rgba(148, 163, 184, 0.16)"
  },
  gold: {
    base:  "#f59e0b",
    light: "#fbbf24",
    dark:  "#92400e",
    tint:  "rgba(245, 158, 11, 0.22)"
  },
  platinum: {
    base:  "#8b5cf6",
    light: "#a78bfa",
    dark:  "#5b21b6",
    tint:  "rgba(139, 92, 246, 0.22)"
  }
};

const SURFACE = {
  bg:     "#0f0f0f",
  card:   "#1a1a1a",
  elev:   "#262626",
  border: "#2d2d2d"
};

// renderGrid(frameFn) calls frameFn(badge, tier, surface) for each badge and
// expects an SVG string. Wraps each into a card with the name + tier label.
function renderGrid(frameFn) {
  const grid = document.getElementById("grid");
  if (!grid) return;
  grid.innerHTML = BADGES.map(b => {
    const t = TIER_COLORS[b.tier];
    const svg = frameFn(b, t, SURFACE);
    const tier = b.tier.charAt(0).toUpperCase() + b.tier.slice(1);
    const cat  = b.cat.charAt(0).toUpperCase() + b.cat.slice(1);
    return `
      <div class="card tier-${b.tier}">
        ${svg}
        <div class="name">${b.name}</div>
        <div class="meta">${tier} · ${cat}</div>
      </div>
    `;
  }).join("");
}

// Subtle placeholder marking where the icon will sit, rendered in tier color.
// Used by each shape so reviewers can focus on the frame itself.
function placeholderSlot(cx, cy, r, color, opacity = 0.35) {
  return `
    <circle cx="${cx}" cy="${cy}" r="${r}" fill="none"
            stroke="${color}" stroke-width="1.4" stroke-dasharray="2 3"
            opacity="${opacity}"/>
  `;
}
