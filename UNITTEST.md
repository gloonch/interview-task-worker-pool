
### This document summarizes the tests added to the project, what behaviors they validate, and how to run them (including race detection).


## How to run tests

Run all tests:
```bash
go test -race ./...
```

Here’s the list of test scenarios covered so far (by package):

## `internal/store/memory`

* **Create + Get (happy path)**

  * Creates a task, assigns an incremental `ID > 0`
  * Forces initial status to `pending` (even if input status is different)
  * `Get(id)` returns the correct task
* **Get (not found)**

  * `Get` on a missing id returns `ok=false`
* **List**

  * After multiple creates, `List` returns the right count and contains the created tasks
* **Fail**

  * `Fail(id, reason)` sets `status=failed` and `Error=reason`
  * Subsequent `Get` reflects the failure
* **Fail (not found)**

  * Failing a missing id returns `ErrNotFound`
* **UpdateStatus**

  * Updates a task’s status and persists it in the store
* **UpdateStatus (not found)**

  * Updating a missing id returns `ErrNotFound`
* **Concurrent Create**

  * Multiple goroutines calling `Create` concurrently → correct final count, no data races (validated with `-race`)

---

## `internal/service`

* **Constructor validation**

  * `New(nil, pool)` returns `ErrStoreNil`
  * `New(store, nil)` returns `ErrPoolNil`
* **CreateTask validation**

  * Empty/whitespace title → `ErrInvalidInput`
  * Ensures store/pool are not called on invalid input
* **CreateTask happy path**

  * Calls `store.Create`
  * Calls `pool.Enqueue` with the created task ID
  * Ensures `CreatedAt` and `WorkDuration` are set (non-zero)
* **CreateTask + PoolFull**

  * `Enqueue` returns `ErrPoolFull`
  * Service marks the task as `failed` via `store.Fail`
  * Returned task has `status=failed` and `error="task pool is full"`
  * Returned error is `ErrPoolFull`
* **CreateTask + PoolClosed**

  * `Enqueue` returns `ErrPoolClosed`
  * Service marks the task as `failed` via `store.Fail`
  * Returned task has `status=failed` and `error="task pool is closed"`
  * Returned error is `ErrPoolClosed`
* **GetTask validation**

  * `id <= 0` returns `ErrInvalidID`

---

## `internal/workerpool`

* **Process single task**

  * Enqueue one task → status transitions `pending → running → done`
* **Overflow / backpressure**

  * With `poolSize=1` and `workers=0`:

    * first enqueue succeeds
    * second enqueue returns `ErrPoolFull`
* **Shutdown drains queued work**

  * Enqueue multiple tasks, call `Shutdown(ctx)`
  * Shutdown waits for workers to finish (no early return)
  * All tasks end in `done`
* **Enqueue after shutdown**

  * After `Shutdown`, `Enqueue` returns `ErrPoolClosed`

---

## `internal/http/handlers` via `httptest`

* **POST /tasks**

  * Happy path returns `201 Created` with `id > 0` and `status=pending`
* **POST /tasks (invalid json)**

  * Malformed JSON returns `400 Bad Request`
* **POST /tasks (invalid input)**

  * Empty title returns `400 Bad Request`
* **GET /tasks/{id}**

  * Existing id returns `200 OK` with the correct task payload
* **GET /tasks/{id} (not found)**

  * Missing id returns `404 Not Found`
* **GET /tasks/{id} (invalid id)**

  * Invalid id (e.g. `0`) returns `400 Bad Request`
* **GET /tasks**

  * Returns `200 OK` and a list containing created tasks
* **POST /tasks (pool full)**

  * When queue is full, returns `503 Service Unavailable`
  * Response body includes `status=failed` and a non-empty `error`
* **POST /tasks (pool closed)**

  * After shutting down the pool, POST returns `503 Service Unavailable`
