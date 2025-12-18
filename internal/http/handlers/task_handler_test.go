package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"interview-task-worker-pool/internal/domain"
	"interview-task-worker-pool/internal/http/dto"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	approuter "interview-task-worker-pool/internal/http"
	"interview-task-worker-pool/internal/http/handlers"
	"interview-task-worker-pool/internal/service"
	"interview-task-worker-pool/internal/store/memory"
	"interview-task-worker-pool/internal/workerpool"
)

func newApp(t *testing.T, poolSize int, workers int) (http.Handler, func()) {
	t.Helper()

	store := memory.New()
	pool := workerpool.New(poolSize, store)
	pool.Start(workers)

	svc, err := service.New(store, pool)
	if err != nil {
		t.Fatalf("service.New err=%v", err)
	}

	h := handlers.New(svc)
	router := approuter.New(h)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = pool.Shutdown(ctx)
	}

	return router, cleanup
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body err=%v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	return rr
}

func doRaw(t *testing.T, h http.Handler, method, path string, raw string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(raw))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	return rr
}

func TestPOST_Tasks_Created(t *testing.T) {
	app, cleanup := newApp(t, 10, 0)
	defer cleanup()

	rr := doJSON(t, app, http.MethodPost, "/tasks", map[string]any{
		"title":       "Buy groceries",
		"description": "Milk",
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d, want %d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}

	var out dto.TaskResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode err=%v", err)
	}

	if out.ID <= 0 {
		t.Fatalf("id=%d, want > 0", out.ID)
	}
	if out.Status != string(domain.StatusPending) {
		t.Fatalf("status=%q, want %q", out.Status, string(domain.StatusPending))
	}
}

func TestPOST_Tasks_InvalidJSON_400(t *testing.T) {
	app, cleanup := newApp(t, 10, 0)
	defer cleanup()

	rr := doRaw(t, app, http.MethodPost, "/tasks", "{bad json}")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestPOST_Tasks_EmptyTitle_400(t *testing.T) {
	app, cleanup := newApp(t, 10, 0)
	defer cleanup()

	rr := doJSON(t, app, http.MethodPost, "/tasks", map[string]any{
		"title":       "   ",
		"description": "x",
	})

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestGET_TaskByID_OK(t *testing.T) {
	app, cleanup := newApp(t, 10, 0)
	defer cleanup()

	create := doJSON(t, app, http.MethodPost, "/tasks", map[string]any{
		"title":       "T1",
		"description": "D1",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}

	var created dto.TaskResponse
	_ = json.NewDecoder(create.Body).Decode(&created)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks/"+strconv.FormatInt(created.ID, 10), nil)
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var out dto.TaskResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode err=%v", err)
	}
	if out.ID != created.ID {
		t.Fatalf("id=%d, want %d", out.ID, created.ID)
	}
}

func TestGET_TaskByID_NotFound_404(t *testing.T) {
	app, cleanup := newApp(t, 10, 0)
	defer cleanup()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks/999999", nil)
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want %d body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
}

func TestGET_TaskByID_InvalidID_400(t *testing.T) {
	app, cleanup := newApp(t, 10, 0)
	defer cleanup()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks/0", nil)
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func TestGET_Tasks_List_OK(t *testing.T) {
	app, cleanup := newApp(t, 10, 0)
	defer cleanup()

	_ = doJSON(t, app, http.MethodPost, "/tasks", map[string]any{
		"title":       "A",
		"description": "x",
	})
	_ = doJSON(t, app, http.MethodPost, "/tasks", map[string]any{
		"title":       "B",
		"description": "y",
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var out []dto.TaskSummaryResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode err=%v", err)
	}
	if len(out) < 2 {
		t.Fatalf("len=%d, want >=2", len(out))
	}
}

func TestPOST_Tasks_PoolFull_503AndFailedTask(t *testing.T) {
	// poolSize=1 workers=0
	//  - POST : fills queue
	//  - POST : error 503 should appear because pool is full
	app, cleanup := newApp(t, 1, 0)
	defer cleanup()

	first := doJSON(t, app, http.MethodPost, "/tasks", map[string]any{
		"title":       "T1",
		"description": "x",
	})
	if first.Code != http.StatusCreated {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}

	second := doJSON(t, app, http.MethodPost, "/tasks", map[string]any{
		"title":       "T2",
		"description": "y",
	})
	if second.Code != http.StatusServiceUnavailable {
		t.Fatalf("second status=%d, want %d body=%s", second.Code, http.StatusServiceUnavailable, second.Body.String())
	}

	var out dto.TaskResponse
	if err := json.NewDecoder(second.Body).Decode(&out); err != nil {
		t.Fatalf("decode err=%v", err)
	}
	if out.Status != string(domain.StatusFailed) {
		t.Fatalf("status=%q, want %q", out.Status, string(domain.StatusFailed))
	}
	if out.Error == "" {
		t.Fatalf("error is empty, want non-empty")
	}
}

func TestPOST_Tasks_PoolClosed_503(t *testing.T) {
	// this test only passes when handler throws 503 error for ErrPoolClosed

	store := memory.New()
	pool := workerpool.New(10, store)
	pool.Start(0)

	svc, _ := service.New(store, pool)
	h := handlers.New(svc)
	app := approuter.New(h)

	// pool shutdown to request after it
	_ = pool.Shutdown(context.Background())

	rr := doJSON(t, app, http.MethodPost, "/tasks", map[string]any{
		"title":       "T",
		"description": "D",
	})

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want %d body=%s", rr.Code, http.StatusServiceUnavailable, rr.Body.String())
	}
}
