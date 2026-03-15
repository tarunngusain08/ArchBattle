# ArchBattle Codebase Remediation Plan

This document outlines issues identified during a codebase review and provides an in-depth plan for fixing them. Issues are grouped by severity and category.

---

## Executive Summary

| Severity | Count | Focus Areas |
|----------|-------|-------------|
| **Critical** | 2 | Rate limiter fail-open, WS client crash on malformed JSON |
| **High** | 4 | Lobby player display, lock ordering, ShareCard validation, Discussion upvote |
| **Medium** | 6 | Input validation, security, error handling |
| **Low** | 7 | Accessibility, API consistency, edge cases |

---

## 1. Critical Issues

### 1.1 Rate Limiter Fail-Open Not Implemented

**Location:** `core/internal/adapter/inbound/http/match_handler.go:62-67`

**Problem:** When Redis fails, `rateLimiter.Allow()` returns `(false, 0, err)`. The handler has a comment "Fail open — do not block the player" but does not set `allowed = true` on error. Users are blocked from queueing when Redis is unavailable.

**Current code:**
```go
allowed, count, err := h.rateLimiter.Allow(r.Context(), key, 5, 24*time.Hour)
if err != nil {
    // Fail open — do not block the player
}
if !allowed {
    writeJSON(w, stdhttp.StatusTooManyRequests, ...)
    return
}
```

**Fix:**
```go
allowed, count, err := h.rateLimiter.Allow(r.Context(), key, 5, 24*time.Hour)
if err != nil {
    allowed = true // Fail open — do not block the player when Redis is down
}
if !allowed {
    ...
}
```

**Effort:** 5 min

---

### 1.2 WS Client Crashes on Malformed JSON

**Location:** `frontend/src/ws/client.ts:43-44`

**Problem:** `JSON.parse(event.data)` can throw if the server sends malformed JSON. An uncaught exception breaks the `onmessage` handler and may prevent further message processing.

**Fix:** Wrap in try/catch:
```typescript
this.socket.onmessage = (event) => {
  try {
    const msg = JSON.parse(event.data) as SocketMessage
    // ... rest of handler
  } catch (e) {
    console.warn('Invalid WS message', e)
  }
}
```

**Effort:** 5 min

---

## 2. High Priority Issues

### 2.1 Lobby Displays UUIDs Instead of Usernames

**Location:** 
- Backend: `core/internal/domain/match/service.go:86, 108`
- Frontend: `frontend/src/stores/matchStore.ts:68-69`, `frontend/src/components/match/LobbyCard.tsx`

**Problem:** `match_created` and `lobby_state` events send `players: playerIDs` (UUID array). The frontend stores and displays these directly. LobbyCard shows raw UUIDs (e.g. `a1b2c3d4-e5f6-...`) instead of usernames.

**Fix options:**

**Option A (Backend):** Include usernames in the payload. The match service already has access to match players. Change the payload to:
```go
Payload: map[string]any{
    "players": []map[string]any{
        {"id": playerID, "username": username},
        ...
    },
    ...
}
```
Requires fetching usernames when broadcasting. The match state store has PlayerIDs; we need a lookup. The gateway has `clients[userID]` with username. When broadcasting to a match, we can build a map of userID -> username from connected clients, or fetch from match_players.

**Option B (Backend):** Add a `player_profiles` field to `match_created` and `lobby_state` that includes `{userId, username}` for each player. The match service's `JoinMatch` and `CreateMatch` flow has access to player data. In `CreateMatch`, we have `playerIDs` from the match record. We need to resolve usernames—either from the match_players table (after AddPlayers) or from the request. The CreateMatchRequest has PlayerProfile with Username. So we can include usernames in the initial match_created. For lobby_state when a new player joins, we need updated player list with usernames. The state store has PlayerIDs. We'd need to look up usernames—e.g. from the match_players table or from the gateway's client map.

**Recommended approach:** In the match service, when publishing `match_created`, we have the match record. After AddPlayers, we could call GetPlayers to get usernames. Or we could store a map at match creation. The simplest: include `player_profiles` in the event payload with `{userId, username}` from match_players. Add a helper to fetch player profiles for a match and include in broadcast.

**Effort:** 1–2 hours

---

### 2.2 Lock Ordering Risk in WS Gateway

**Location:** `core/internal/adapter/inbound/ws/handler.go:140-159`

**Problem:** `handleJoinMatch` acquires `g.mu.RLock()` then `g.loopMu.Lock()`. If another goroutine (e.g. in BroadcastToMatch or CancelMatchLoop) acquires `loopMu` first and then needs `mu`, deadlock can occur.

**Fix:** Establish a consistent lock order. Convention: always acquire `loopMu` before `mu` when both are needed, or vice versa. Audit all sites that take both locks and ensure consistent order. Alternatively, restructure to avoid holding both locks simultaneously (e.g. copy data under one lock, release, then acquire the other).

**Effort:** 30–60 min

---

### 2.3 ShareCard Ignores Invalid Query Params

**Location:** `core/internal/adapter/inbound/http/daily_handler.go:71-74`

**Problem:** `strconv.Atoi` and `ParseFloat` errors are ignored. Invalid `score`, `correct`, `total`, `percentile` yield 0, producing incorrect or misleading share card text. Unauthenticated endpoint; anyone can generate arbitrary share cards.

**Fix:**
1. Validate query params: return 400 if missing or invalid.
2. Consider requiring auth for ShareCard (optional; may be intentional for sharing).
3. Add optional rate limiting for unauthenticated endpoints.

```go
score, err := strconv.Atoi(r.URL.Query().Get("score"))
if err != nil || score < 0 {
    writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid score"})
    return
}
// Similar for correct, total, percentile
```

**Effort:** 20 min

---

### 2.4 Discussion Upvote Has No Error Handling

**Location:** `frontend/src/pages/Discussion.tsx:31-35`

**Problem:** `upvoteEntry(date, entry.id)` is called without `await` or `.catch()`. Failed upvotes are silent; user gets no feedback.

**Fix:**
```tsx
onClick={async () => {
  try {
    await upvoteEntry(date, entry.id)
    onUpvote()
  } catch (e) {
    setError?.('Failed to upvote')
  }
}}
```
Add local error state or toast for failed upvotes.

**Effort:** 15 min

---

## 3. Medium Priority Issues

### 3.1 QuestionHandler Dispute Ignores readJSON Errors

**Location:** `core/internal/adapter/inbound/http/question_handler.go:37-38`

**Problem:** `_ = readJSON(r, &req)` ignores decode errors. Invalid JSON produces empty `req` and `req.Reason` is empty string; may cause unexpected behavior.

**Fix:** Check and handle error:
```go
if err := readJSON(r, &req); err != nil {
    writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
    return
}
```

**Effort:** 5 min

---

### 3.2 WS CheckOrigin Allows All Origins

**Location:** `core/internal/adapter/inbound/ws/gateway.go:108-111`

**Problem:** `CheckOrigin` returns `true` for all origins. Enables CSRF/abuse from arbitrary origins.

**Fix:** Use `AllowedOrigins` from config. The gateway must receive config (or an AllowedOrigins slice). Inject into NewGateway and validate:
```go
CheckOrigin: func(r *stdhttp.Request) bool {
    origin := r.Header.Get("Origin")
    if origin == "" {
        return true
    }
    for _, allowed := range g.allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}
```

**Effort:** 30 min (requires config wiring)

---

### 3.3 Discussion Text Length Unbounded

**Location:** `core/internal/domain/discussion/service.go`, `model.go`

**Problem:** No max length for `ReasoningText`, `AlternativeText`, `SurpriseText`. Large payloads can cause DoS or resource issues.

**Fix:** Add validation in CreateRequest:
```go
const MaxReasoningLen = 2000
const MaxAlternativeLen = 1000
const MaxSurpriseLen = 1000
if len(req.ReasoningText) > MaxReasoningLen { return nil, ErrTextTooLong }
// Similar for others
```

**Effort:** 15 min

---

### 3.4 Registration Validation Minimal

**Location:** `core/internal/domain/auth/service.go:36-39`

**Problem:** Only checks: username/email not empty, password ≥ 8 chars. No username length, email format, or password complexity.

**Fix:** Add validation:
- Username: 3–30 chars, alphanumeric + underscore
- Email: format validation (regex or simple check)
- Password: optional complexity (uppercase, number, symbol)

**Effort:** 30 min

---

### 3.5 Daily Submit Answers Map Keys Not Validated

**Location:** `core/internal/adapter/inbound/http/daily_handler.go:43-67`

**Problem:** `Answers` map keys are not validated as UUIDs. Invalid keys could be stored or ignored silently.

**Fix:** When iterating submissions, validate each key is a valid UUID before processing. Ignore invalid keys and optionally log.

**Effort:** 15 min

---

### 3.6 Discussion Upvote Ignores Date in URL

**Location:** `core/internal/adapter/inbound/http/discussion_handler.go:90-106`

**Problem:** Upvote handler does not validate that the entry belongs to the given date. The entry_id is globally unique, so the date is redundant for lookup, but the URL is misleading. Low impact—upvote still works correctly.

**Fix (optional):** Validate that the entry's challenge_date matches the URL date. If not, return 404. Requires fetching the entry first.

**Effort:** 20 min

---

## 4. Low Priority Issues

### 4.1 Accessibility: Login Inputs Lack Labels

**Location:** `frontend/src/pages/Login.tsx:21-25`

**Fix:** Add `aria-label` or associate `<label htmlFor="...">` with inputs. Ensure focus and visible labels.

**Effort:** 10 min

---

### 4.2 Accessibility: Upvote Button

**Location:** `frontend/src/pages/Discussion.tsx:32-39`

**Fix:** Add `aria-label="Upvote"` and optionally `aria-pressed` for state.

**Effort:** 5 min

---

### 4.3 Accessibility: QuestionCard Option Buttons

**Location:** `frontend/src/components/match/QuestionCard.tsx`

**Fix:** Add `aria-label` for option buttons and `aria-pressed` for selected state.

**Effort:** 10 min

---

### 4.4 handleAnswerSubmit Question Not Found

**Location:** `core/internal/adapter/inbound/ws/handler.go:174-179`

**Problem:** If `GetByID` returns error, we return early. Verify there is no path where `question` is nil but we use it. Current code returns `err` on GetByID error, so we never reach `question.GetCorrectAnswers()`. **Status:** Likely OK; verify.

**Effort:** 5 min (verification)

---

### 4.5 Learning Summary Error Ignored

**Location:** `core/internal/domain/match/service.go:311-312`

**Problem:** `summary, _ = s.summaries.RequestLearningSummary(...)` ignores error. Summary is optional; failure is acceptable.

**Fix (optional):** Log the error for observability.

**Effort:** 2 min

---

### 4.6 HTTP 30s Timeout for Long-Running Routes

**Location:** `core/internal/adapter/inbound/http/router.go:63`

**Problem:** AI summary generation or other slow operations may exceed 30s.

**Fix:** Exclude specific routes (e.g. `/admin/daily-challenge/summary`) from the timeout, or increase timeout for admin routes.

**Effort:** 15 min

---

### 4.7 UserMatchHistory Query for Multi-Player Matches

**Location:** `core/internal/adapter/outbound/postgres/match_repo.go:152-154`

**Problem:** The subquery `(SELECT username FROM match_players WHERE match_id = um.match_id AND user_id != $1 LIMIT 1)` returns only one opponent. For 2v2 or 3-player matches, we show "Solo" or one opponent. May be acceptable for "last 10 matches" display.

**Fix (optional):** For multi-player matches, concatenate opponent names or show "vs Team" / "vs 2 players".

**Effort:** 30 min

---

## 5. Implementation Order

| Phase | Items | Est. Time |
|-------|-------|-----------|
| **Phase 1** | 1.1 Rate limiter fail-open, 1.2 WS JSON parse, 2.3 ShareCard validation, 2.4 Upvote error handling, 3.1 readJSON | 1 hour |
| **Phase 2** | 2.1 Lobby usernames, 2.2 Lock ordering | 2–3 hours |
| **Phase 3** | 3.2 CheckOrigin, 3.3 Discussion text length, 3.4 Registration validation | 1.5 hours |
| **Phase 4** | 3.5 Daily submit validation, 4.1–4.3 Accessibility | 1 hour |
| **Phase 5** | 4.4–4.7 Optional fixes | 1 hour |

**Total:** ~6–7 hours

---

## 6. Testing Recommendations

1. **Rate limiter:** Simulate Redis failure; verify users can still queue.
2. **WS client:** Send malformed JSON from a mock server; verify no crash.
3. **Lobby:** Verify usernames display after Phase 2.
4. **ShareCard:** Send invalid query params; verify 400 response.
5. **Discussion** upvote: Simulate network failure; verify error feedback.
6. **Lock ordering:** Run load tests; monitor for deadlocks.

---

## 7. Appendix: File Reference

| Issue | File |
|-------|------|
| 1.1 | `core/internal/adapter/inbound/http/match_handler.go` |
| 1.2 | `frontend/src/ws/client.ts` |
| 2.1 | `core/internal/domain/match/service.go`, `frontend/src/stores/matchStore.ts`, `frontend/src/components/match/LobbyCard.tsx` |
| 2.2 | `core/internal/adapter/inbound/ws/handler.go`, `gateway.go` |
| 2.3 | `core/internal/adapter/inbound/http/daily_handler.go` |
| 2.4 | `frontend/src/pages/Discussion.tsx` |
| 3.1 | `core/internal/adapter/inbound/http/question_handler.go` |
| 3.2 | `core/internal/adapter/inbound/ws/gateway.go` |
| 3.3 | `core/internal/domain/discussion/service.go` |
| 3.4 | `core/internal/domain/auth/service.go` |
| 3.5 | `core/internal/adapter/inbound/http/daily_handler.go` |
| 3.6 | `core/internal/adapter/inbound/http/discussion_handler.go` |
