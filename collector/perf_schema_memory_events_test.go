package collector

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/smartystreets/goconvey/convey"
)

func TestScrapePerfMemoryEvents(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()

	columns := []string{
		"EVENT_NAME",
		"SUM_NUMBER_OF_BYTES_ALLOC",
		"SUM_NUMBER_OF_BYTES_FREE",
		"CURRENT_NUMBER_OF_BYTES_USED",
	}

	rows := sqlmock.NewRows(columns).
		AddRow("memory/innodb/event1", "1001", "500", "501").
		AddRow("memory/innodb/event2", "2002", "1000", "1002").
		AddRow("memory/sql/event1", "30", "4", "26")
	mock.ExpectQuery(sanitizeQuery(perfMemoryEventsQuery)).WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		if err = (ScrapePerfMemoryEvents{}).Scrape(context.Background(), db, ch, log.NewNopLogger()); err != nil {
			panic(fmt.Sprintf("error calling function on test: %s", err))
		}
		close(ch)
	}()

	metricExpected := []MetricResult{
		{labels: labelMap{"event_name": "memory/innodb/event1"}, value: 1001, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"event_name": "memory/innodb/event1"}, value: 500, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"event_name": "memory/innodb/event1"}, value: 501, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"event_name": "memory/innodb/event2"}, value: 2002, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"event_name": "memory/innodb/event2"}, value: 1000, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"event_name": "memory/innodb/event2"}, value: 1002, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"event_name": "memory/sql/event1"}, value: 30, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"event_name": "memory/sql/event1"}, value: 4, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"event_name": "memory/sql/event1"}, value: 26, metricType: dto.MetricType_GAUGE},
	}
	convey.Convey("Metrics comparison", t, func() {
		for _, expect := range metricExpected {
			got := readMetric(<-ch)
			convey.So(got, convey.ShouldResemble, expect)
		}
	})

	// Ensure all SQL queries were executed
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}