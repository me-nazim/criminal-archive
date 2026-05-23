# UI / UX Design — Tansiq Information Portal

> Status: **DRAFT (planning round 1)** · Last updated: 2026-05-23

This document defines the design language, the component library, and a
page-by-page specification for the v1 frontend. It is paired with the API
contract in [`API_SPEC.md`](./API_SPEC.md).

---

## 1. Design principles

1. **Archive, not a feed.** The portal is a permanent reference, not a
   social timeline. Visual rhythm rewards depth, not scroll-velocity.
2. **Journalism, not entertainment.** The tone is closer to ProPublica,
   The Intercept, Wikipedia, and the Internet Archive than to social media.
   No clickbait, no infinite-scroll dopamine loops.
3. **Bengali first, English peer.** The default visual hierarchy assumes
   Bengali script. We pick fonts and line-heights that sit comfortably with
   Bengali ascenders and English ascenders side by side.
4. **Truth signals everywhere.** Verified status, source links, evidence
   badges, and submission dates are foregrounded so the reader can decide
   whether to trust an item.
5. **Minimum chrome.** Heavy on typography and content; light on borders,
   shadows, gradients, and decorative imagery. Premium = restraint.
6. **Responsive by default.** Reading must work on a 360 px wide phone over
   2G. Decorative imagery is loaded last.
7. **No dark patterns.** No newsletter pop-ups, no "register to read",
   no consent walls beyond what privacy law actually requires.

## 2. Brand & visual identity

### 2.1 Name

- **Bengali:** তানসিক ইনফরমেশন পোর্টাল
- **English:** Tansiq Information Portal (short: *Tansiq*)

### 2.2 Logotype

- A serif wordmark using **Tiro Bangla** for the Bengali name and
  **Inter Tight** (or **Inter** in display weight) for the English name.
- A square monogram "ত" (or stylised "T") in white on `brand-500`,
  used as the favicon and small avatars.

### 2.3 Colour palette

We pair a near-black archival ink with one warm accent and four functional
status colours. The palette is already configured in
[`frontend/tailwind.config.js`](../frontend/tailwind.config.js) and may be
refined as we build.

| Token | Hex | Usage |
| -- | -- | -- |
| `ink-50`  | `#f6f7f9` | Page background |
| `ink-100` | `#eceef2` | Subtle dividers, hover states |
| `ink-200` | `#d5dae3` | Default borders |
| `ink-600` | `#4d586d` | Secondary text |
| `ink-900` | `#0f1320` | Primary text, headlines, dark surfaces |
| `brand-500` | `#e8501f` | Primary accent (links, active nav, CTA) |
| `brand-600` | `#c63d14` | Pressed / active state |
| `success-600` | `#0f7a4f` | Verified badge, success toasts |
| `warning-600` | `#a8730c` | "Pending verification" badge |
| `danger-600` | `#b91c1c` | Errors, takedown notice |
| `info-600` | `#1d4ed8` | Informational badges, links inside body copy (alt) |

> Note: `success-600`, `warning-600`, `danger-600`, `info-600` are not yet
> in the Tailwind config; they are added as part of Phase 1.

### 2.4 Typography

| Role | Font | Notes |
| -- | -- | -- |
| Body (Bengali) | **Hind Siliguri** 400/500/700 | Clean, legible at small sizes |
| Body (Latin)   | **Inter** 400/500/700 | Pairs well with Hind Siliguri |
| Display (Bengali) | **Tiro Bangla** | Editorial headlines |
| Display (Latin)   | **Inter Tight** or **Inter** display cut | Editorial headlines |
| Mono | **JetBrains Mono** or system monospace | Case numbers, code, hashes |

Font sizes (4-pt scale):

| Token | Size | Line height | Use |
| -- | -- | -- | -- |
| `text-xs` | 12 | 16 | Meta, captions |
| `text-sm` | 14 | 20 | Secondary body |
| `text-base` | 16 | 24 | Default body |
| `text-lg` | 18 | 28 | Lead paragraphs |
| `text-xl` | 20 | 28 | Card titles |
| `text-2xl` | 24 | 32 | Section heads |
| `text-3xl` | 30 | 36 | Page titles |
| `text-4xl` | 36 | 40 | Hero titles |
| `text-5xl` | 48 | 56 | Hero (desktop) |

### 2.5 Spacing & layout

- 4 px base grid. Tailwind defaults align.
- Max content width:
  - Reading layouts (case detail, person profile, blog): **`max-w-3xl`** (768 px).
  - List / index pages: **`max-w-7xl`** (1280 px).
  - Admin app: full width with internal `max-w-screen-2xl`.
- Section vertical rhythm: `py-12` mobile / `py-20` desktop.

### 2.6 Iconography

- [`lucide-react`](https://lucide.dev) — outline style, 1.5 px stroke.
- Custom evidence-type icons (image, video, document, audio) live in
  `frontend/src/components/icons/`.

### 2.7 Imagery rules

- All photographs of identifiable people are tagged at upload with one of:
  `consented_public`, `redacted`, `accused`, `news_screenshot`. The renderer
  uses this tag to apply blurring or a watermark when needed.
- Victim photos default to `is_anonymous=true` and are never rendered for
  the public, even if a file is attached.
- Cover images use a 16:9 crop with focal-point metadata.

---

## 3. Component library

The following components are owned by the design system and live in
`frontend/src/components/ui/`. They are styled with Tailwind and have
explicit, type-safe props.

### 3.1 Atoms

| Component | Variants | Notes |
| -- | -- | -- |
| `Button` | `primary`, `secondary`, `ghost`, `danger` × `sm`, `md`, `lg` | Always renders a `<button>` or `<a>` based on `as` prop. |
| `IconButton` | square, round | Has accessible label. |
| `Badge` | `neutral`, `success`, `warning`, `danger`, `info` | Used for status, kinds, severity. |
| `Tag` | clickable / static | Used for free-form tags on cases. |
| `Avatar` | photo, monogram fallback, sizes 24/32/40/64 | |
| `Spinner` | 3 sizes | |
| `ProgressBar` | determinate / indeterminate | |
| `Tooltip` | radix-based wrapper | |

### 3.2 Form controls

| Component | Notes |
| -- | -- |
| `TextField` | label, helper, error, prefix/suffix slots |
| `TextArea` | autosize variant |
| `RichTextEditor` | based on Tiptap; bn + en aware; image paste blocked |
| `Select` / `Combobox` | searchable; supports async loading (used for persons, locations) |
| `LocationCascade` | composite of 4 `Combobox`es: country → division → district → upazila. Falls back to a single free-text field for non-BD countries. |
| `DatePicker` | bn + en localised; supports DOB (with year-only), incident date (with time) |
| `FileDropzone` | drag-drop; multi-file; hooks into the presign + upload flow |
| `Checkbox`, `Radio`, `Switch` | |
| `FormError` | renders zod / API field errors uniformly |

### 3.3 Composites

| Component | Notes |
| -- | -- |
| `CaseCard` | Title, location, date, crime-type badge, cover image, view-count |
| `CaseListItem` | Compact row variant for dense lists |
| `PersonCard` | Photo, name (or "anonymous"), occupation, case count |
| `AttachmentTile` | Thumbnail, filename, size, kind badge, copy URL button |
| `EvidenceGallery` | Lightbox + grid of public attachments |
| `Timeline` | Vertical timeline of `case_timeline` entries |
| `NewsSourceList` | Renders external links with favicons + archived link |
| `RoleBadge` | Visual role marker for users |
| `StatusPill` | Submission state pill (`draft` / `pending_review` / etc.) |
| `LanguageSwitcher` | Already implemented in Phase 0 |
| `Header`, `Footer`, `Breadcrumbs` | Already partially implemented |
| `EmptyState`, `ErrorState` | Standard placeholders |
| `Pagination` (cursor) | Hides "page numbers", shows prev/next + "loading more" |

---

## 4. Information architecture

```
/                                 Home
/cases                            Browse cases (filters in URL)
/cases/:slug                      Case detail   (canonical URL)
/cases/by-number/:case_number     Alias        (TIP-YYYY-NNNNN)
/persons                          Browse persons
/persons/:slug                    Person profile
/search?q=...                     Cross-resource search

/about                            Static
/methodology                      "How we verify" page
/contact                          Contact / takedown form

/login
/register
/forgot-password
/reset-password?token=...

/me                               Account profile (logged in)
/me/cases                         My submissions
/me/cases/new                     Submit form (multi-step)
/me/cases/:id/edit                Edit own draft

/admin                            Admin shell (gated)
/admin/dashboard
/admin/users
/admin/users/:id
/admin/cases                      All cases (any status)
/admin/cases/:id                  Editor with internal notes / hidden attach
/admin/persons
/admin/persons/:id
/admin/verification               Queue + assignments
/admin/verification/:caseId
/admin/crime-types
/admin/audit-log
/admin/settings                   Super-admin only
```

URL grammar:
- Slugs are lower-kebab Latin: `rape-incident-savar-2026-04`. Bengali slugs
  are not used in URLs (they encode poorly and reduce share-ability).
- The `case_number` route is a permanent alias and never changes.

---

## 5. Page specifications

For each page we specify: **purpose**, **above-the-fold layout**, **data
sources**, **states**, and **CTAs**.

### 5.1 Home (`/`)

**Purpose:** establish trust, point visitors at recent cases, guide new
contributors to the submission flow.

**Layout:**

```
┌────────────────────────────────────────────────────────────┐
│  Header  [logo] [Home Cases Persons Submit] [bn|en] Login  │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  H1 (display, bn primary):                                 │
│  "প্রতিটি ঘটনার সত্য তথ্য, সকলের জন্য উন্মুক্ত।"            │
│                                                            │
│  Lead (text-lg, ink-600): subtitle paragraph              │
│                                                            │
│  [ ব্রাউজ করুন ]   [ তথ্য জমা দিন ]                        │
│                                                            │
│  Trust strip:  📁 X published   👥 Y profiles   ✅ verified│
│                                                            │
├────────────────────────────────────────────────────────────┤
│  ## সাম্প্রতিক ঘটনা                                        │
│  [CaseCard] [CaseCard] [CaseCard]            (3 columns)   │
├────────────────────────────────────────────────────────────┤
│  ## কীভাবে যাচাই হয়                                        │
│   3-step infographic: Submit → Verify → Publish            │
├────────────────────────────────────────────────────────────┤
│  Footer                                                    │
└────────────────────────────────────────────────────────────┘
```

**Data:** `GET /api/v1/cases?limit=6&sort=published_desc`, plus a small
`stats` endpoint (or compute counts in page).

### 5.2 Cases list (`/cases`)

**Purpose:** discoverable, filterable archive of all published cases.

**Layout:**

```
┌────────────────────────────────────────────────────────────┐
│  Search bar (q)  [ Filters ▾ ]                             │
├──────────────┬─────────────────────────────────────────────┤
│ Filters     │ Sort: Recently published ▾   N results       │
│ ─────────── │                                              │
│ Country     │ ┌────────┐ ┌────────┐ ┌────────┐             │
│ Division    │ │ Case   │ │ Case   │ │ Case   │             │
│ District    │ │ Card   │ │ Card   │ │ Card   │             │
│ Upazila     │ └────────┘ └────────┘ └────────┘             │
│ Crime type  │ ┌────────┐ ┌────────┐ ┌────────┐             │
│ Year        │ │  ...   │ │  ...   │ │  ...   │             │
│ Tags (chips)│ └────────┘ └────────┘ └────────┘             │
│             │                  [ Load more ]               │
└─────────────┴──────────────────────────────────────────────┘
```

- On mobile, filters collapse into a sticky bottom-sheet.
- All filter state lives in the URL query string for shareability.
- Cards show: cover image (or placeholder), case_number, title (bn/en
  switching), location, incident date, crime-type badge, view count.

**States:** loading skeleton (6 cards) · empty (illustration + CTA to
clear filters) · error (retry).

### 5.3 Case detail (`/cases/:slug`)

**Purpose:** a single page that lets a reader form an opinion based on the
evidence.

**Layout (reading column, max-w-3xl):**

```
┌────────────────────────────────────────────────────────────┐
│  Breadcrumb: Cases / Dhaka / Savar / TIP-2026-00045        │
│                                                            │
│  H1 case title (bn/en)                                     │
│  Meta row: case_number · crime type · location · date      │
│  Status pill: ✅ Verified · Published 2026-04-25            │
│                                                            │
│  Cover image (16:9)                                        │
│                                                            │
│  Summary paragraph                                         │
│                                                            │
│  ## বিস্তারিত                                              │
│  long-form description                                     │
│                                                            │
│  ## অভিযুক্ত ও ভিকটিম                                       │
│  PersonCards row (links to person profiles)                │
│                                                            │
│  ## টাইমলাইন                                                │
│  Timeline component                                        │
│                                                            │
│  ## প্রমাণ ও সংযুক্তি                                       │
│  EvidenceGallery (public attachments only)                 │
│  [ সব ডাউনলোড করুন (zip) ]                                  │
│                                                            │
│  ## সংবাদসূত্র                                              │
│  NewsSourceList                                            │
│                                                            │
│  ## পরিবর্তনের ইতিহাস (optional, lightweight)               │
└────────────────────────────────────────────────────────────┘
```

- Sticky mini ToC on the right on desktop ≥ `lg`.
- A floating "report inaccuracy" button lives bottom-right.
- Bengali-first; if `_en` exists a tab at the top of the H1 toggles language
  for that case alone.

### 5.4 Person profile (`/persons/:slug`)

```
┌─────────────────────────────────────────────────────────┐
│  [Avatar 96]  Name (bn/en)                              │
│               Aliases · Occupation · Designation        │
│               Location                                  │
│                                                         │
│  ## পরিচয়                                              │
│  public_bio                                             │
│                                                         │
│  ## যেসব ঘটনায় যুক্ত (N)                                │
│  CaseCard list                                          │
└─────────────────────────────────────────────────────────┘
```

If `is_anonymous`: avatar is a generic silhouette, name is rendered as
"Anonymous victim", DOB and address are hidden. Linked cases are still
shown.

### 5.5 Submit a case (`/me/cases/new`)

A 4-step wizard with a sticky progress bar and "save draft" on every step.

1. **মৌলিক তথ্য** — title, summary, incident date/time, location cascade,
   crime type, tags.
2. **ব্যক্তি যুক্ত করুন** — search existing persons or create new
   victim/accused/witness rows. Inline create dialog.
3. **প্রমাণ আপলোড** — file dropzone with progress. Each file gets a
   sequence number preview (`TIP-YYYY-XXXXX_evidence_03.jpg`). Mark
   each as `public` / `hidden` (only contributors+ can mark hidden;
   visibility caps at admin review anyway).
4. **প্রিভিউ ও জমা** — read-only summary + "জমা দিন" button.

A draft can be left at any step; revisiting `/me/cases/:id/edit` restores
it.

### 5.6 Auth pages

- `/login` — single column, max-w-md card, primary CTA, link to register.
- `/register` — same shape; on submit shows a success page explaining the
  pending-approval flow.
- `/forgot-password` — single email input.
- `/reset-password` — token-bound new-password form.

### 5.7 Admin shell (`/admin/*`)

- Two-column layout: persistent left nav (Dashboard, Approvals, Cases,
  Persons, Verification, Crime types, Audit, Settings), main content right.
- Top bar shows the impersonation banner if the admin is viewing as
  another role (post-v1 feature, listed here for foresight).

#### `/admin/dashboard`

Simple grid of metric cards:
- Pending users
- Cases in `pending_review`
- Cases `in_verification`
- Cases ready to publish (`approved`)
- Verifications I own (if also a verifier)
- Recent audit log entries

#### `/admin/users`

Table: name, email, role, status, created, actions (approve/reject/suspend/role).
Filters: status, role, search.

#### `/admin/cases/:id`

Same as the public detail page **plus** internal notes editor, hidden
attachments tile-grid, verifier assignment widget, publish controls,
audit log slice for this case.

#### `/admin/verification`

Two views: my queue, all assignments. Each item shows due-since,
attached-evidence summary, link to a verifier-mode case page.

### 5.8 Error pages

- `/404` — minimal, with a search suggestion.
- `/403` — explains why access is blocked (`account_pending`,
  `forbidden`, etc).
- `/500` — generic, with a request id for support.

---

## 6. Responsive strategy

Breakpoints (Tailwind defaults):

| | width | layout shifts |
| -- | -- | -- |
| `sm`  | 640 px | nav becomes inline; case-card grid 1 col |
| `md`  | 768 px | nav inline; cards 2 col; reading col still single |
| `lg`  | 1024 px | cards 3 col; sticky ToC appears on case detail |
| `xl`  | 1280 px | full max-w-7xl on list pages |
| `2xl` | 1536 px | admin shell can use ≤ 1408 px content |

Mobile-first, touch-target ≥ 44 px, no hover-only affordances on tap
devices.

---

## 7. Accessibility

- All interactive elements reachable by keyboard with a visible focus
  ring (`outline-2 outline-offset-2 outline-brand-500`).
- Colour contrast: body ≥ 7:1, large text ≥ 4.5:1.
- All images have `alt` text. Decorative images use `alt=""`.
- Forms: labels are `<label htmlFor>`-bound; errors are programmatically
  associated via `aria-describedby`.
- Live regions (`aria-live="polite"`) for toasts and async progress.
- The language switcher updates `<html lang>` and triggers a re-render
  of any localized number / date formatters.

## 8. Motion

- We use `framer-motion` sparingly: only for modal/dialog enter, toast
  enter/exit, and lightbox transitions.
- Page transitions are intentionally absent — the archive feel demands
  instant navigation.
- All motion respects `prefers-reduced-motion`.

## 9. Internationalisation rules

- Translation keys are flat-namespaced: `nav.home`, `submit.step.basics`.
- Numbers and dates use `Intl.NumberFormat`/`Intl.DateTimeFormat` with the
  active locale.
- Bengali numerals are rendered when the active locale is `bn`. We support
  toggling Latin numerals via a per-user setting (post-v1).
- Mixed-script content is rendered with `font-feature-settings` left at
  defaults; we rely on the chosen fonts to handle bi-directional spacing.

## 10. Empty / loading / error UX checklist

For every list and detail screen we ship at minimum:

- A **loading skeleton** that matches the final layout (no spinners on
  primary content).
- An **empty state** with a one-line headline and a helpful CTA.
- An **error state** with a short explanation and a retry button.

These are codified as `EmptyState` / `ErrorState` components and used
consistently.

## 11. Open questions for the design phase

1. Logo direction — are we going with a serif "ত" monogram, or a custom
   wordmark? (Needs a designer pass before launch.)
2. Cover-image policy — auto-pick first attachment, or always require a
   curator-chosen image? (Current plan: auto-pick with admin override.)
3. Map view of cases by district — in v1 or post-v1? (Recommendation:
   post-v1, after we have ≥ 30 cases to make the map worth rendering.)
4. Should the public download zip include news source PDFs (when we
   archive them) or only original attachments? (Default: only originals.)
