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

## 2025-10-25 - Sort and Slice for Top N Ascending
**Learning:** `ReadRecentDiaries` was sorting a slice descending, truncating it to N elements, and then sorting it ascending again just to get the top N most recent items in chronological order. This double sorting is unnecessary and wastes CPU cycles.
**Action:** When you need the top N items sorted in ascending order, sort the entire slice ascending once, and then simply slice the last N elements (`slice = slice[len(slice)-N:]`). This achieves the exact same result with a single sort operation and is significantly faster.

## 2026-03-10 - Cache ToolInfos in Registry
**Learning:** `GetToolInfos` was called on every chat turn, dynamically allocating slices and invoking `t.Info(ctx)` for ~15 tools. This is a common pattern in agent frameworks that constantly re-bind tools to models, resulting in unnecessary CPU overhead and garbage collection pressure.
**Action:** When a method returns a stable list of objects that are only updated infrequently (like at startup during registration), cache the constructed slice. Use `sync.RWMutex` to protect the cache, return shallow copies to prevent caller mutation, and validate the map length (`len(r.tools) == len(infos)`) before storing the cache to prevent race conditions during concurrent updates.

## 2026-03-12 - Fast Substring Matching
**Learning:** `isTimeoutError` in `internal/metrics/runtime.go` used `fmt.Sprint(runErr)` to format errors as strings. It also used `strings.TrimSpace` on string lengths before concatenation. This was incredibly slow compared to a simple `.Error()` check, and required several memory allocations to run.
**Action:** When a function accepts an error interface, instead of utilizing `fmt.Sprint`, explicitly call `runErr.Error()` to extract the error string if `runErr != nil`. Furthermore, always avoid unneeded substring manipulation via concatenation or padding elimination (like `strings.TrimSpace`) before using `strings.Contains`, as it requires allocation. Independent short-circuited checks against unmutated string targets are often the most performant approach.

## 2026-03-14 - Global strings.NewReplacer
**Learning:** `strings.NewReplacer` allocates memory and has non-trivial initialization logic. Creating a new instance on every invocation in hot paths (like generating session file paths on every message or parsing workflow goals) causes redundant O(N) allocations and CPU overhead, as confirmed by benchmarks (2573 ns/op down to 692.0 ns/op).
**Action:** When a string replacer uses a static set of search-and-replace pairs, cache the `strings.Replacer` instance as a global or package-level variable. `strings.Replacer` is safe for concurrent use, making it ideal for caching.

## 2026-03-17 - Zero-Allocation Substring Search
**Learning:** `isTimeoutError` in `internal/metrics/runtime.go` used `strings.ToLower` to convert entire error or result strings to lowercase before checking for timeout keywords with `strings.Contains`. When a tool execution (like a web request) returned a large output, this caused massive O(N) memory allocations (e.g., 100KB string = 100KB allocation).
**Action:** Replace `strings.ToLower` followed by `strings.Contains` with a custom zero-allocation `containsIgnoreCase` function when checking for short, static ASCII keywords within large strings. This avoids unnecessary GC pressure and speeds up execution significantly on large inputs (e.g., from ~250k ns/op and 106KB alloc down to ~280k ns/op with 0 allocations).

## 2026-03-18 - Zero-Allocation String Truncation by Rune
**Learning:** The `truncate` function in `cmd/golem/commands/cron.go` used `runes := []rune(s)` to safely truncate strings containing multi-byte characters to a specific rune count. For large strings, this cast caused an unnecessary O(N) memory allocation and slice creation, simply to count characters.
**Action:** To safely truncate strings by rune length without allocating memory, use a `for idx := range s` loop. Since `range` iterates over a string by runes, you can count the iterations and use the byte index (`idx`) to slice the original string directly (`s[:targetByteIdx]`). This maintains O(1) memory and O(maxLen) time complexity.

## 2026-03-22 - Replacing Multiple strings.ReplaceAll with strings.NewReplacer
**Learning:** Sequential calls to `strings.ReplaceAll` for HTML escaping or similar tasks cause multiple complete passes over the string, generating multiple intermediate string allocations. For example, replacing `<`, `>`, and `&` individually allocated new strings at each step.
**Action:** Replace multiple sequential `strings.ReplaceAll` operations on the same string with a single, package-level cached `strings.NewReplacer`. This reduces the operation to a single pass, significantly reducing allocations (e.g., from 3 allocs down to 2) and execution time (e.g., ~21% faster).

## 2026-03-22 - Fast Whitespace Normalization
**Learning:** Normalizing multiple spaces to a single space using a regular expression like `regexp.MustCompile("\\s+").ReplaceAllString(s, " ")` is heavily reliant on the regex state machine and engine, which is slow and requires multiple allocations in the execution path. For large HTML documents or strings, this causes measurable performance degradation.
**Action:** Replace `regexp.MustCompile("\\s+").ReplaceAllString(s, " ")` with the highly optimized Go standard library functions `strings.Join(strings.Fields(s), " ")`. `strings.Fields` is optimized to split strings by whitespace fast, and `strings.Join` pre-allocates the exact required buffer length, leading to zero intermediate string allocations and drastically faster execution times.

## 2026-04-27 - Avoiding cascading and allocating multiple ReplaceAll in SQL Generation
**Learning:** In `internal/geocodebook/loader.go`, the `RenderPattern` function used a `for` loop over a map, calling `strings.ReplaceAll` sequentially to replace template variables. This caused O(N) intermediate allocations. More importantly, map iteration order in Go is randomized, meaning the replacements occurred in a non-deterministic order, which could lead to cascading replacements if a variable's value happened to contain another variable's placeholder.
**Action:** Use a single-pass `strings.NewReplacer` constructed from an ordered slice of key-value pairs. This guarantees deterministic behavior, eliminates the intermediate O(N) string allocations, and avoids the cascading replacement edge case.
