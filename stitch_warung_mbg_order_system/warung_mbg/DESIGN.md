---
name: Warung MBG
colors:
  surface: '#faf8ff'
  surface-dim: '#d2d9f4'
  surface-bright: '#faf8ff'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f2f3ff'
  surface-container: '#eaedff'
  surface-container-high: '#e2e7ff'
  surface-container-highest: '#dae2fd'
  on-surface: '#131b2e'
  on-surface-variant: '#3c4a42'
  inverse-surface: '#283044'
  inverse-on-surface: '#eef0ff'
  outline: '#6c7a71'
  outline-variant: '#bbcabf'
  surface-tint: '#006c49'
  primary: '#006c49'
  on-primary: '#ffffff'
  primary-container: '#10b981'
  on-primary-container: '#00422b'
  inverse-primary: '#4edea3'
  secondary: '#0058be'
  on-secondary: '#ffffff'
  secondary-container: '#2170e4'
  on-secondary-container: '#fefcff'
  tertiary: '#b91a24'
  on-tertiary: '#ffffff'
  tertiary-container: '#ff7a73'
  on-tertiary-container: '#79000e'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#6ffbbe'
  primary-fixed-dim: '#4edea3'
  on-primary-fixed: '#002113'
  on-primary-fixed-variant: '#005236'
  secondary-fixed: '#d8e2ff'
  secondary-fixed-dim: '#adc6ff'
  on-secondary-fixed: '#001a42'
  on-secondary-fixed-variant: '#004395'
  tertiary-fixed: '#ffdad7'
  tertiary-fixed-dim: '#ffb3ad'
  on-tertiary-fixed: '#410004'
  on-tertiary-fixed-variant: '#930013'
  background: '#faf8ff'
  on-background: '#131b2e'
  surface-variant: '#dae2fd'
typography:
  display-lg:
    fontFamily: Plus Jakarta Sans
    fontSize: 40px
    fontWeight: '700'
    lineHeight: 48px
    letterSpacing: -0.02em
  display-lg-mobile:
    fontFamily: Plus Jakarta Sans
    fontSize: 30px
    fontWeight: '700'
    lineHeight: 36px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Plus Jakarta Sans
    fontSize: 28px
    fontWeight: '600'
    lineHeight: 36px
    letterSpacing: -0.015em
  headline-md:
    fontFamily: Plus Jakarta Sans
    fontSize: 22px
    fontWeight: '600'
    lineHeight: 28px
    letterSpacing: -0.01em
  headline-sm:
    fontFamily: Plus Jakarta Sans
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
    letterSpacing: -0.005em
  body-lg:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
    letterSpacing: 0em
  body-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
    letterSpacing: 0em
  body-sm:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 16px
    letterSpacing: 0.01em
  label-lg:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
    letterSpacing: 0.01em
  label-md:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
    letterSpacing: 0.02em
  label-sm:
    fontFamily: Inter
    fontSize: 10px
    fontWeight: '700'
    lineHeight: 14px
    letterSpacing: 0.04em
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  gutter: 1rem
  gutter-desktop: 1.5rem
  margin: 1rem
  margin-tablet: 1.5rem
  margin-desktop: 2.5rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 1rem
  space-lg: 1.5rem
  space-xl: 2rem
---

## Brand & Style

This design system blends **Light Glassmorphism** with refined **Contemporary Minimalism**, tailored specifically for modern digital culinary commerce and quick-service ordering.

### Brand Personality & Emotional Impact
- **Fresh & Organic Appetite:** Fresh emerald greens evoke appetizing quality, culinary hygiene, vitality, and trust.
- **Fluid & Effortless Flow:** Airy, luminous panels with backdrop blurs provide an uncluttered browsing experience that lets food photography take center stage.
- **Tactile Softness:** Pill-shaped interactions, deep corner radii (24px to 32px), and low-contrast borders evoke a polished, premium native app feel.

### Design Aesthetic
The visual language operates on floating translucent cards, subtle hairline glass boundaries, layered light diffusion, and soft ambient glows. Dense data tables and harsh dividers are rejected in favor of breathable whitespace, clear visual grouping, and fluid navigation chrome reminiscent of modern ambient OS interfaces.

## Colors

The palette revolves around appetizing emerald tones paired with crisp architectural neutrals and clear functional status accents.

### Palette Construction
- **Primary (`#10B981`):** Active ordering triggers, cart confirmations, checkout highlights, and primary brand badges. Supported by surface tints: `#ECFDF5` (emerald-50) and `#D1FAE5` (emerald-100) for container backgrounds and active pill states.
- **Secondary (`#3B82F6`):** Secondary information tags, delivery/logistics metadata, and supportive action prompts.
- **Tertiary (`#EF4444`):** Store closure indicators ("Tutup"), out-of-stock badges, and cancellation paths.
- **Neutrals & Surfaces:**
  - Canvas Base: `#F8FAFC` to `#F1F5F9` (cool, clean foundation).
  - Floating Surfaces: `#FFFFFF` at 85%–95% opacity with backdrop blur.
  - Text Primary: `#0F172A` (dense slate for maximum legibility).
  - Text Secondary: `#334155` (supporting details, prices, notes).
  - Text Tertiary: `#64748B` (metadata, placeholder labels).

## Typography

The typography pairs **Plus Jakarta Sans** for warm, geometric headlines with **Inter** for neutral, utilitarian microcopy and dense menu reading.

### Hierarchy & Treatment
- **Display & Headlines:** Set in Plus Jakarta Sans with subtle negative tracking (`-0.01em` to `-0.02em`) to maintain tight, editorial titles across merchant headers and featured dish banners.
- **Body & Pricing:** Set in Inter for maximum optical precision in numeric prices (e.g., `Rp 24.000`) and customizable dietary ingredient descriptions.
- **Status Badges & Micro Labels:** Rendered uppercase or semi-bold with positive tracking (`+0.02em` to `+0.04em`) to maintain visual legibility on small-form pill indicators.

## Layout & Spacing

The layout is built upon an 8pt base spacing rhythm within a fluid-adaptive containment grid.

### Layout Philosophy
- **Mobile Handheld (320px – 640px):** 4-column fluid layout with sticky floating action bars (cart summary, bottom nav pill), 16px lateral canvas margin, and 12px card gaps.
- **Tablet (641px – 1024px):** 8-column layout with 24px margins; displays dual-pane views (category rail alongside product listing).
- **Desktop (1025px+):** 12-column layout centered at max-width 1280px with 40px outer margins, allowing persistent sticky sidebars for orders and order tracking.
- **Rhythm Rules:** Vertical stacking within food cards uses `space-xs` (4px) and `space-sm` (8px). Structural section separations utilize `space-lg` (24px) to `space-xl` (32px).

## Elevation & Depth

Visual hierarchy uses frosted glassmorphism, translucency, and diffuse ambient shadows to create depth without visual noise.

### Surface Tiers & Blurs
- **Level 0 (Canvas):** Solid light wash (`#F8FAFC`).
- **Level 1 (Card Resting):** White at 90% opacity (`rgba(255, 255, 255, 0.90)`), with `backdrop-filter: blur(12px)`, edged with a hairline border (`rgba(209, 250, 229, 0.6)` or `rgba(226, 232, 240, 0.8)`). Shadow: `0 4px 16px -2px rgba(15, 23, 42, 0.04)`.
- **Level 2 (Hover / Active Cards):** White at 96% opacity, `backdrop-filter: blur(16px)`, shadow: `0 10px 24px -4px rgba(16, 185, 129, 0.08), 0 4px 12px -2px rgba(15, 23, 42, 0.05)`.
- **Level 3 (Floating Bars & Modals):** White at 85% opacity, `backdrop-filter: blur(20px)`, subtle emerald perimeter glow, shadow: `0 20px 32px -6px rgba(15, 23, 42, 0.10)`.

### Border Rationale
Avoid opaque heavy borders. All card boundaries use 1px borders with 40%–60% alpha to preserve the frosted edge.

## Shapes

The design system uses generous, organic roundedness to communicate approachability and modern tactile feedback.

### Radius Scaling
- **Standard UI Elements (`rounded-lg` / 16px):** Small buttons, input containers, quantity counters, and badge wrappers.
- **Cards & Food Containers (`rounded-xl` / 24px):** Product cards, promo banners, and modal containers.
- **Hero & Floating Nav (`rounded-2xl` / 32px):** Top hero panels and bottom floating action docks.
- **Full Pill (`rounded-full`):** Category filters, operational status badges ("Buka / Tutup"), and primary action pills.

## Components

### Buttons
- **Primary:** Solid `#10B981` background with crisp `#FFFFFF` text. On hover, a subtle emerald glow (`box-shadow: 0 8px 20px -4px rgba(16, 185, 129, 0.40)`). Rounded-full or rounded-xl.
- **Secondary / Soft:** Tinted `#ECFDF5` background with `#047857` label, no border, 100% tactile scale down (0.98x) on active tap.
- **Ghost:** Transparent background with `#334155` text, switching to `#F1F5F9` on hover.

### Status Badges ("Buka / Tutup")
- **Buka (Open):** Pill shape with light emerald fill (`#ECFDF5`), deep emerald text (`#065F46`), and a 6px pulsing emerald indicator dot (`#10B981`).
- **Tutup (Closed):** Pill shape with light crimson fill (`#FEF2F2`), deep red text (`#991B1B`), and a static muted red dot (`#EF4444`).

### Food Cards
- **Structure:** Floating frosted container (`rgba(255, 255, 255, 0.92)`). High-resolution food photography at the top with a 16px inner radius.
- **Metadata:** Dish name in Plus Jakarta Sans (`headline-sm`), brief description in Inter (`body-sm`), price in bold slate (`#0F172A`), accompanied by an inline circular "+" quick-add action button tinted in emerald.

### Category Chips & Filters
- **Resting:** Frosted white with subtle slate outline (`rgba(226, 232, 240, 0.8)`), text `#334155`.
- **Selected:** Solid emerald green (`#10B981`) text or tinted emerald backing (`#ECFDF5`) with an active emerald indicator line.

### Inputs & Steppers
- **Search Field:** Frosted bar with search icon in `#64748B`, placeholder in `#94A3B8`, focused ring using `rgba(16, 185, 129, 0.3)`.
- **Quantity Selector:** Compact pill container holding a minus icon, numeric value, and plus icon, encased in subtle `#F1F5F9` fill.

### Floating Cart Dock
- Bottom-docked floating panel with high-blur glass (`backdrop-filter: blur(24px)`), featuring order item count, total price summary, and a high-contrast emerald checkout action trigger.