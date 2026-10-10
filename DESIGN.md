# Design — Project Identity

> This document is project-long-lived. Tokens are not changed without
> the Architect's approval. Developers MUST use these tokens
> instead of improvising their own colors/spacings.

## Style Direction

Calm, light professional "workshop office" interface in the spirit of Linear/Stripe: near-white surfaces, a single cool petrol accent, quiet borders and a colour-coded status system that carries the whole order workflow (requested → confirmed → in progress → done → picked up).

## Colors

- `--color-bg`: **#F6F7F9**
- `--color-surface`: **#FFFFFF**
- `--color-surface-sunken`: **#EEF1F4**
- `--color-fg`: **#182230**
- `--color-muted`: **#5A6675**
- `--color-border`: **#DDE2E8**
- `--color-border-strong`: **#C2CAD3**
- `--color-accent`: **#0F646B**
- `--color-accent-hover`: **#0B4F55**
- `--color-accent-active`: **#093F44**
- `--color-accent-soft`: **#E3F0F1**
- `--color-on-accent`: **#FFFFFF**
- `--color-error`: **#B3261E**
- `--color-error-soft`: **#FBEAE8**
- `--color-success`: **#15803D**
- `--color-success-soft`: **#E4F4E8**
- `--color-warning`: **#B45309**
- `--color-warning-soft`: **#FDF1DC**
- `--color-info`: **#2563EB**
- `--color-info-soft`: **#E6EEFE**
- `--color-status-requested`: **#5A6675**
- `--color-status-requested-soft`: **#EEF1F4**
- `--color-status-confirmed`: **#2563EB**
- `--color-status-confirmed-soft`: **#E6EEFE**
- `--color-status-in-progress`: **#B45309**
- `--color-status-in-progress-soft`: **#FDF1DC**
- `--color-status-done`: **#15803D**
- `--color-status-done-soft`: **#E4F4E8**
- `--color-status-picked-up`: **#0F646B**
- `--color-status-picked-up-soft`: **#E3F0F1**
- `--color-overlay`: **rgba(24,34,48,0.45)**

## Typography

- `font_family`: system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif
- `font_family_mono`: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace
- `heading_weight`: 600
- `body_weight`: 400
- `label_weight`: 500
- `size-display`: 32px / 40px, weight 600
- `size-h1`: 24px / 32px, weight 600
- `size-h2`: 20px / 28px, weight 600
- `size-h3`: 16px / 24px, weight 600
- `size-body`: 15px / 24px, weight 400
- `size-small`: 13px / 20px, weight 400
- `size-label`: 12px / 16px, weight 500, letter-spacing 0.02em, uppercase for table headers
- `numerals`: font-variant-numeric: tabular-nums on every money, quantity, hour and timestamp value
- `loading`: No webfonts are fetched; only this system stack is used so that no third-party network resource is loaded (AC-28).

## Spacing Scale

- `--space-0`: 4px
- `--space-1`: 8px
- `--space-2`: 12px
- `--space-3`: 16px
- `--space-4`: 24px
- `--space-5`: 32px
- `--space-6`: 48px

## Border-Radii

- `--radius-sm`: 4px
- `--radius-md`: 8px
- `--radius-lg`: 12px
- `--radius-pill`: 999px

## Components

### Button

Base: min-height 44px (touch target), padding 12px 20px, radius md (8px), font 15px weight 500, gap 8px between icon and label, no text wrapping, transition 120ms ease-out on background/border/shadow. Variants: primary = bg accent #0F646B, text on-accent #FFFFFF; secondary = bg surface #FFFFFF, text fg, border 1px border-strong #C2CAD3; ghost = transparent bg, text accent, no border; danger = bg error #B3261E, text #FFFFFF. States (primary): default bg accent; hover bg accent-hover #0B4F55; active bg accent-active #093F44 (+1px inset shadow); focus-visible 2px outline accent with 2px offset; disabled opacity 0.5, cursor not-allowed, no hover change. States (secondary): hover bg surface-sunken #EEF1F4, border border-strong; active bg #E4E8EC. Loading: label stays, spinner 16px left of label, button disabled. Below 640px a primary action in a form goes full width; button groups stack with 8px gap. Icon-only buttons are 44x44px with an aria-label.

### Field (Input / Textarea / Select)

Label above the control, 13px weight 500, fg, margin-bottom 6px, always visible (no placeholder-only labels). Control: width 100%, min-height 44px, padding 12px, radius sm (4px), border 1px border #DDE2E8, bg surface, text 15px fg; textarea min-height 96px, resize vertical. Placeholder in muted #5A6675. States: default border; hover border border-strong; focus border accent + 2px accent-soft ring, no native outline; disabled bg surface-sunken, text muted; invalid border error #B3261E, helper text below in 13px error. Validation timing: errors appear only after the field was filled and blurred, or after a submit attempt — never on first render. Helper/error line reserves 20px height so layout does not jump. Optional unit suffix (e.g. 'h', 'km') is muted and right-aligned inside the control. Select uses the same metrics with a chevron icon and 12px right padding offset.

### Card / Panel

bg surface #FFFFFF, radius lg (12px), border 1px border #DDE2E8, shadow 0 1px 2px rgba(24,34,48,0.04), padding 24px desktop / 16px mobile, gap 16px between inner blocks. Card header: 16px/24px weight 600 title on the left, optional action (ghost Button or status badge) right-aligned, separated from the body by a 1px border line with 16px padding above and below. Cards in a list get 16px vertical gap; on mobile card padding drops to 16px and full card width is used.

### StatusBadge

Pill (radius pill), height 24px, padding 4px 10px, font 12px weight 500, leading colour dot 6px. One fixed mapping — angefragt: text/border status-requested #5A6675 on status-requested-soft #EEF1F4; bestätigt: #2563EB on #E6EEFE; in Arbeit: #B45309 on #FDF1DC; fertig: #15803D on #E4F4E8; abgeholt: #0F646B on #E3F0F1. Never communicate status by colour alone — the German label ('angefragt', 'bestätigt', 'in Arbeit', 'fertig', 'abgeholt') is always visible. A dimmed variant (opacity 0.45) marks statuses the order has not reached yet in the progress indicator.

### OrderProgress (status stepper)

Horizontal on desktop, vertical below 640px. Five steps 'angefragt → bestätigt → in Arbeit → fertig → abgeholt'. Reached steps: dot 10px filled with the status colour, label 13px weight 500 fg; current step additionally gets a 2px accent-soft ring and weight 600; future steps: dot border 1px border-strong, label muted. Connector line 2px, reached segments in the status colour, rest in border #DDE2E8. Every step has 44px minimum touch/line height; the whole stepper is read-only (status changes go through an explicit Button).

### DataTable (Auftragsliste)

Column header row: 12px uppercase weight 500 muted, 8px/12px padding, sticky on scroll, bottom border 1px border. Rows: min-height 56px, 12px/16px padding, bottom border 1px border, hover bg surface-sunken, whole row clickable with a focus ring. Cells: Kennzeichen in mono tabular-nums weight 500, status as StatusBadge, Wunschtermin and Erstellzeit right-aligned tabular-nums. Below 640px the table reflows into stacked Cards, one per order, with a label–value grid (each field label in 12px muted above a 15px value) and the status badge in the card header — no horizontal scrolling of a shrunken table.

### FilterBar

Sits above the order list: a status filter (Select with 'Alle Status' plus the five statuses) and a licence-plate search input with a magnifier icon, side by side on desktop (12px gap), stacked full width below 640px. Active filters shown as removable pills (radius pill, bg accent-soft, text accent) with a 24px 'x' target next to a 'Filter zurücksetzen' ghost Button. Search matches partial plates and shows an empty state when nothing matches.

### Timeline (Statusverlauf)

Vertical list, newest first. Each entry: left rail 2px border with a 8px dot in the new status colour, 16px indent for the content, timestamp 13px muted tabular-nums on the top line, below it 'Alter Status → Neuer Status' with both values as small status labels. Entries separated by 16px. Read-only, no interaction; when empty, the EmptyState component replaces it with 'Noch keine Statuswechsel'.

### StatePanel (loading / empty / error)

One component for all three non-content states, used on every page. Layout: centered block, max-width 480px, padding 32px, gap 8px: icon 24px muted, title 16px weight 600 fg, description 15px muted, optional action Button (ghost or primary). loading: three skeleton bars (radius sm, bg surface-sunken, 100%/80%/60% width, 1.4s pulse) — skeletons mirror the real layout, not a spinner-only page. empty: descriptive text plus next step (e.g. 'Noch keine Aufträge — hier erscheint die erste Anfrage aus dem Kundenbereich.'). error: error-soft background, 4px left border error, message plus a 'Erneut versuchen' Button; server messages are shown with the API error code in a muted 13px line underneath.

### AppShell / Navigation

Top header, height 56px mobile / 64px desktop, bg surface, bottom border 1px border, sticky. Left: wordmark 16px weight 600 with a 20px petrol mark. Center/right on desktop: a two-segment area switch ('Kundenbereich' / 'Werkstatt') as a segmented control — radius md, 1px border, active segment bg accent-soft text accent, inactive text muted, each segment min-height 44px; below 640px the switch moves into a full-width row under the header. Workshop-wide navigation (Dashboard, Aufträge) appears as a 44px-high tab row with an accent underline for the active tab. Right side: the staff e-mail (13px muted) and a 'Abmelden' ghost Button. Footer: 13px muted, always links to 'Impressum' and 'Datenschutzerklärung' on every visible page (AC-27), minimum 44px link tap height.

### FormActions

Bottom of every form: primary submit on the left of the group (or right-aligned on desktop, full width on mobile), secondary 'Abbrechen' as ghost Button, 12px gap. While submitting, the submit button shows its loading state and all fields are disabled to prevent double submission. After a failed submit the first invalid field receives focus and the inline errors plus a summary Alert appear.

### ValueDisplay (money, date, hours)

The single formatting component for every value the product shows — IDs and licence plates use the mono stack with tabular-nums, all numbers right-aligned in tables. Order number: as issued, e.g. 'A-2024-0042'. Licence plate: as stored but normalised to uppercase, e.g. 'B-AB 1234'. Date: 'TT.MM.JJJJ' (14.03.2025). Time: 'HH:MM' 24-hour, local time (Europe/Berlin), stored and transferred as UTC. Date + time together: 'TT.MM.JJJJ, HH:MM' (14.03.2025, 09:30). Duration/labour: one decimal plus unit, e.g. '1,5 h'. Mileage: integer plus 'km', thousands separated, e.g. '128.400 km'. Money: always cents-derived, German formatting with comma decimals and dot thousands separator, currency suffix, e.g. '1.234,56 €'; negative amounts are never shown. Tax rate: '19 %'. Quantity with 1–1000 separator: '2' or '1.200'. Missing values show as '—' in muted, never as an empty cell. A 'noch nicht verfügbar' state (e.g. invoice not yet created) is rendered as muted text, e.g. 'Rechnung noch nicht verfügbar', never as 0,00 €.

### InvoiceSummary

Card with a bordered table of positions: three columns 'Bezeichnung / Menge / Einzelpreis' on desktop with a right-aligned 'Gesamt' column, collapsing to 'Bezeichnung' as a heading with 'Menge × Einzelpreis' as a muted sub-line and the line total right-aligned below 640px. Below the positions a right-aligned, 16px-separated amount block: 'Nettosumme', 'MwSt. (19 %)' and 'Bruttosumme' — the last line 20px weight 600 with a 1px top border. Every amount uses ValueDisplay money formatting with tabular-nums. Read-only, no print/PDF action (out of scope).

### KPICard (Dashboard)

Grid of five cards: 'angefragt', 'bestätigt', 'in Arbeit', 'heute fertig', 'Umsatz laufender Monat'. Value in 32px weight 600 tabular-nums, label in 12px uppercase muted above it, optional muted sub-line (e.g. '× 120,00 € Stundensatz' is not shown; instead the revenue card notes 'aus gestellten Rechnungen'). Desktop: 5 columns / tablet 2–3 / mobile 1 column, 16px gap. Each KPI card carries the matching status colour as a 3px top border, except revenue which uses accent. A KPI link (e.g. requested orders) prefilters the order list.

### Notice / Alert

Inline block for confirmations and API errors: radius md, padding 12px 16px, 1px border, 4px left border in the semantic colour, icon 16px plus 15px text. error: bg error-soft, border error, text #7A1A14; success: bg success-soft, border success, text #0F5B2B; warning: bg warning-soft, border warning; info: bg info-soft, border info. The API error code is shown once as a muted 13px line, e.g. 'Code: duplicate_licence_plate'. Alerts are announced via role=status / role=alert for screen readers and never contain personal data from logs.

### LoginCard

Centered card, max-width 400px, 32px page padding, radius lg. Title 'Anmeldung Werkstatt' 20px weight 600, fields E-Mail and Passwort (password input with a 44px show/hide toggle as ghost icon Button), primary submit 'Anmelden' full width, 24px below the fields. On 429 the Notice reads 'Zu viele Versuche. Bitte später erneut versuchen.' with the retry window; on 401 a generic 'E-Mail oder Passwort ist falsch.' — never revealing which field was wrong. Autofocus on the e-mail field, submit via Enter, no client-side rate-limit UI beyond that message.

## Layout Principles

- Container: max-width 1120px, centered, page padding 16px on mobile and 24px from 640px up; the whole product must be usable down to 360px width with no horizontal page scroll.
- Breakpoints: 640px (mobile → tablet: tables become cards, forms and filter bars stack, primary buttons go full width), 1024px (tablet → desktop: two-column layouts, 5-column KPI grid, inline nav). Mobile-first, built only from the 4/8/12/16/24/32/48px scale.
- Page skeleton on every page: sticky header → optional area switch/tabs → h1 (24px weight 600) with a one-line muted description → content → footer with Impressum and Datenschutzerklärung (AC-27). Vertical rhythm: 32px between page sections, 16px between cards in a group, 8px between a label and its value.
- Forms: single column, max-width 560px, field width 100%, 16px vertical gap between fields, one primary action per form; groups of related fields (customer data / vehicle data / appointment) are separated by 24px and an optional 14px weight 600 sub-heading. Never more than one column of inputs on mobile.
- The customer area and the workshop area are visually identical (same tokens, same components) and differ only in navigation and content — the segmented control in the header is the single switch, so users do not have to relearn the UI.
- One formatting rule for every value the product displays: order number 'A-2024-0042', licence plate uppercase 'B-AB 1234', date 'TT.MM.JJJJ' (14.03.2025), time 'HH:MM' local 24-hour (stored UTC), combined 'TT.MM.JJJJ, HH:MM', duration '1,5 h', mileage '128.400 km', money '1.234,56 €' (cents-derived, German separators), tax '19 %'. Empty values always '—' in muted; a not-yet-existing invoice is text, never '0,00 €'.
- Every page implements four visible states: loading (skeleton mirroring the layout), empty (explanation plus next step), error (message plus retry), and content — never an empty white page (AC-21). Every visible button either performs its action or is disabled and labelled 'noch nicht verfügbar'; no dead buttons.
- Accessibility and privacy by construction: text contrast at least 4.5:1 on bg and surface, all interactive targets at least 44x44px, visible focus ring (2px accent, 2px offset) on every focusable element, status never encoded by colour alone, icons carry aria-labels, and no external font, image or script is ever requested — everything ships inside the bundle (AC-28).
- Data density stays calm: at most one accent-filled primary button per view, borders before shadows, no gradients, no decorative imagery; the order status colour is the only strong colour on the workshop screens so that the workflow remains readable at a glance.
