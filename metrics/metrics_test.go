package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCounter(t *testing.T) {
	registry := New()
	counter := NewCounter("test_counter", "Test counter", []string{"label"})
	registry.MustRegister(counter.vec)

	counter.Inc("a")
	counter.Add(5, "b")

	h := registry.Handler()
	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assertContains(t, body, "test_counter")
}

func TestGauge(t *testing.T) {
	registry := New()
	gauge := NewGauge("test_gauge", "Test gauge", []string{"label"})
	registry.MustRegister(gauge.vec)

	gauge.Set(10, "a")
	gauge.Inc("a")
	gauge.Dec("a")
	gauge.Add(2.5, "a")

	h := registry.Handler()
	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code)
	assertContains(t, rec.Body.String(), "test_gauge")
}

func TestHistogram(t *testing.T) {
	registry := New()
	hist := NewHistogram("test_hist", "Test histogram", []string{"label"}, []float64{1, 5, 10})
	registry.MustRegister(hist.vec)

	hist.Observe(2, "a")
	hist.Observe(7, "a")
	hist.Observe(15, "a")

	h := registry.Handler()
	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code)
	assertContains(t, rec.Body.String(), "test_hist")
}

func TestSummary(t *testing.T) {
	registry := New()
	sum := NewSummary("test_summary", "Test summary", []string{"label"}, nil)
	registry.MustRegister(sum.vec)

	sum.Observe(1, "a")
	sum.Observe(2, "a")
	sum.Observe(3, "a")

	h := registry.Handler()
	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code)
	assertContains(t, rec.Body.String(), "test_summary")
}

func TestHandler(t *testing.T) {
	registry := New()
	counter := NewCounter("my_test_counter", "Test counter", nil)
	registry.MustRegister(counter.vec)

	// Increment the counter so it appears in metrics output
	counter.Inc()

	h := registry.Handler()
	requireNotNil(t, h)

	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assertContains(t, body, "my_test_counter")
}

func TestHandlerDefault(t *testing.T) {
	defaultRegistry = New()

	h := HandlerDefault()
	requireNotNil(t, h)

	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code)
}

func TestNewHistogramDefaultBuckets(t *testing.T) {
	hist := NewHistogram("test_hist_buckets", "Test histogram", nil, nil)
	requireNotNil(t, hist)
	requireNotNil(t, hist.vec)

	hist.Observe(0.5)
	hist.Observe(5)
	hist.Observe(50)
}

func TestNewSummaryDefaultObjectives(t *testing.T) {
	sum := NewSummary("test_summary_obj", "Test summary", nil, nil)
	requireNotNil(t, sum)
	requireNotNil(t, sum.vec)

	sum.Observe(1)
	sum.Observe(2)
	sum.Observe(3)
}

func TestMustRegister(t *testing.T) {
	registry := New()
	counter := NewCounter("reg_counter", "Registered counter", nil)
	assertNotPanics(t, func() { registry.MustRegister(counter.vec) })

	gauge := NewGauge("reg_gauge", "Registered gauge", nil)
	assertNotPanics(t, func() { registry.MustRegister(gauge.vec) })

	hist := NewHistogram("reg_hist", "Registered histogram", nil, nil)
	assertNotPanics(t, func() { registry.MustRegister(hist.vec) })

	sum := NewSummary("reg_summary", "Registered summary", nil, nil)
	assertNotPanics(t, func() { registry.MustRegister(sum.vec) })
}

func TestCounterVec(t *testing.T) {
	registry := New()
	counter := NewCounter("test_counter_total", "Test counter", []string{"method", "status"})
	registry.MustRegister(counter.vec)

	counter.Inc("GET", "200")
	counter.Inc("POST", "201")
	counter.Inc("GET", "404")

	h := registry.Handler()
	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code)
	assertContains(t, rec.Body.String(), "test_counter_total")
}

func TestGaugeVec(t *testing.T) {
	registry := New()
	gauge := NewGauge("test_gauge_vec", "Test gauge", []string{"node"})
	registry.MustRegister(gauge.vec)

	gauge.Set(10, "node1")
	gauge.Set(20, "node2")

	h := registry.Handler()
	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assertEqual(t, http.StatusOK, rec.Code)
	assertContains(t, rec.Body.String(), "test_gauge_vec")
}
