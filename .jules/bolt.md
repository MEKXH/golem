## 2026-02-21 - Async Metrics Persistence
**Learning:** The application was performing synchronous file I/O on every tool execution, channel send, and memory recall. This was a significant bottleneck and caused race conditions where concurrent operations could overwrite the metrics file.
**Action:** Implemented an asynchronous buffered persistence mechanism for `RuntimeMetrics`. It now aggregates changes in memory and flushes to disk periodically (every 5s) or on shutdown. Future metrics or logging systems should follow this pattern to avoid blocking the main execution loop.

## 2026-02-21 - Caching Static Context Files
**Learning:** The agent was re-reading 5-8 static configuration files (IDENTITY.md, etc.) and scanning the skills directory on every single interaction turn (LLM request), causing unnecessary disk I/O and latency.
**Action:** Implemented a caching mechanism in ContextBuilder for the base system prompt parts. The cache is invalidated only when file modification tools (write_file, etc.) are executed, ensuring both performance and correctness.

## 2026-02-23 - Selective Cache Invalidation
**Learning:** The previous caching mechanism invalidated the entire system prompt cache whenever *any* file was modified by the agent. This was inefficient for long coding sessions where the agent primarily modifies source code, which doesn't affect the static system prompt.
**Action:** Refined the cache invalidation logic in `ContextBuilder` to check the path of the modified file. The cache is now only cleared if the modified file is one of the base context files (e.g., IDENTITY.md) or resides in the `skills/` directory. This significantly reduces I/O overhead during development tasks.

## 2026-03-09 - Session Manager Lock Contention
**Learning:** The `SessionManager.GetOrCreate` method was acquiring a full write lock (`m.mu.Lock()`) on the entire `Manager` even for retrieving existing sessions. In a multi-channel/server environment where the agent processes messages concurrently, this causes significant lock contention because the vast majority of calls hit existing sessions.
**Action:** Implemented a double-checked locking pattern using a fast-path read lock (`m.mu.RLock()`). The write lock is now only acquired when an actual cache miss occurs (i.e., when creating a new session from disk or memory).

## 2026-03-09 - Memory Context Recall Loop
**Learning:** The memory context recall function `RecallContext` iterated through diary items and called functions that executed `strings.ToLower(content)` inside loops over keywords. For large files or large amounts of memory, this caused many unnecessary lowercasing allocations which hindered performance.
**Action:** Lifted `strings.ToLower()` calls outside of `containsAnyKeyword` and `extractKeywordExcerpt` loops in `RecallContext`. This computes the lowercased strings once per log content and eliminates nested lowercasing calls.

## 2026-03-10 - Buffered Disk I/O for Session Appends
**Learning:** The `SessionManager.Append` method was executing unbuffered JSON encoding directly to a file descriptor. When writing multiple messages in one go (such as user input, tool responses, and agent output), this caused a burst of disk I/O syscalls proportional to the number of objects, severely impacting throughput and generating unnecessary CPU wakeups.
**Action:** Always wrap unbuffered file descriptors with `bufio.NewWriter` before passing them to `json.NewEncoder` (or similar serializers), followed by an explicit `Flush()`. This batches writes into a single syscall, drastically improving performance for multiple small appends.

## 2026-03-10 - Fast Date Validation in Memory Recall
**Learning:** The memory context recall function `collectDiaryFiles` was using `time.Parse` to validate if a file name was a properly formatted date (`YYYY-MM-DD`). Since this function is called on *every* file in the memory directory on every single interaction turn or memory read, `time.Parse`'s expensive allocations and boundary checks were causing significant overhead (~118ns per check).
**Action:** Replaced `time.Parse` with a lightweight, manual `isValidDate` function that performs a simple length and character check (taking ~12ns per check). Avoid using `time.Parse` solely for validating simple machine-generated string patterns in high-frequency loops.

## 2026-03-10 - Zero-Allocation UTF8 Counting
**Learning:** Checking the length of a string in characters via `len([]rune(text))` works but allocates a new slice, causing O(N) memory overhead and GC pressure in hot paths (like keyword tokenization in memory recall).
**Action:** Use `utf8.RuneCountInString(text)` from the `unicode/utf8` package for a fast, zero-allocation way to count characters.

## 2026-03-10 - Optimized Slice Sorting
**Learning:** `ReadRecentDiaries` was blindly double-sorting (descending then ascending) regardless of the number of diary files, wasting CPU cycles on unnecessary sorts when the total file count was already below the target `limit`.
**Action:** Always wrap truncation and double-sorting in a `len(items) > limit` check. When the limit is not exceeded, a single ascending sort is sufficient.
