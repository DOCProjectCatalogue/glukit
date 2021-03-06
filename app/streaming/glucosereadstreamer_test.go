package streaming_test

import (
	"github.com/alexandre-normand/glukit/app/apimodel"
	"github.com/alexandre-normand/glukit/app/bufio"
	"github.com/alexandre-normand/glukit/app/glukitio"
	. "github.com/alexandre-normand/glukit/app/streaming"
	"log"
	"testing"
	"time"
)

type glucoseWriterState struct {
	total      int
	batchCount int
	writeCount int
	batches    map[int64][]apimodel.GlucoseRead
}

type statsGlucoseReadWriter struct {
	state *glucoseWriterState
}

func NewGlucoseWriterState() *glucoseWriterState {
	s := new(glucoseWriterState)
	s.batches = make(map[int64][]apimodel.GlucoseRead)

	return s
}

func NewStatsGlucoseReadWriter(s *glucoseWriterState) *statsGlucoseReadWriter {
	w := new(statsGlucoseReadWriter)
	w.state = s

	return w
}

func (w *statsGlucoseReadWriter) WriteGlucoseReadBatch(p []apimodel.GlucoseRead) (glukitio.GlucoseReadBatchWriter, error) {
	log.Printf("WriteGlucoseReadBatch with [%d] elements: %v", len(p), p)
	dayOfGlucoseReads := []apimodel.DayOfGlucoseReads{apimodel.NewDayOfGlucoseReads(p)}

	return w.WriteGlucoseReadBatches(dayOfGlucoseReads)
}

func (w *statsGlucoseReadWriter) WriteGlucoseReadBatches(p []apimodel.DayOfGlucoseReads) (glukitio.GlucoseReadBatchWriter, error) {
	log.Printf("WriteGlucoseReadBatches with [%d] batches: %v", len(p), p)
	for _, dayOfData := range p {
		log.Printf("Persisting batch with start date of [%v]", dayOfData.Reads[0].GetTime())
		w.state.total += len(dayOfData.Reads)
		w.state.batches[dayOfData.Reads[0].GetTime().Unix()] = dayOfData.Reads
	}

	log.Printf("WriteGlucoseReadBatches with total of %d", w.state.total)
	w.state.batchCount += len(p)
	w.state.writeCount++

	return w, nil
}

func (w *statsGlucoseReadWriter) Flush() (glukitio.GlucoseReadBatchWriter, error) {
	return w, nil
}

func TestWriteOfDayGlucoseReadBatch(t *testing.T) {
	state := NewGlucoseWriterState()
	w := NewGlucoseStreamerDuration(NewStatsGlucoseReadWriter(state), apimodel.DAY_OF_DATA_DURATION)

	ct, _ := time.Parse("02/01/2006 15:04", "18/04/2014 00:00")

	for i := 0; i < 25; i++ {
		readTime := ct.Add(time.Duration(i) * time.Hour)
		w, _ = w.WriteGlucoseRead(apimodel.GlucoseRead{apimodel.Time{apimodel.GetTimeMillis(readTime), "America/Montreal"}, apimodel.MG_PER_DL, float32(i)})
	}

	if state.total != 24 {
		t.Errorf("TestWriteOfDayGlucoseReadBatch failed: got a total of %d but expected %d", state.total, 24)
	}

	if state.batchCount != 1 {
		t.Errorf("TestWriteOfDayGlucoseReadBatch failed: got a batchCount of %d but expected %d", state.batchCount, 1)
	}

	if state.writeCount != 1 {
		t.Errorf("TestWriteOfDayGlucoseReadBatch failed: got a writeCount of %d but expected %d", state.writeCount, 1)
	}
}

func TestWriteOfDayGlucoseReadBatchesInSingleCall(t *testing.T) {
	state := NewGlucoseWriterState()
	w := NewGlucoseStreamerDuration(NewStatsGlucoseReadWriter(state), apimodel.DAY_OF_DATA_DURATION)

	ct, _ := time.Parse("02/01/2006 15:04", "18/04/2014 00:00")

	reads := make([]apimodel.GlucoseRead, 25)

	for i := 0; i < 25; i++ {
		readTime := ct.Add(time.Duration(i) * time.Hour)
		reads[i] = apimodel.GlucoseRead{apimodel.Time{apimodel.GetTimeMillis(readTime), "America/Montreal"}, apimodel.MG_PER_DL, float32(i)}
	}

	w, _ = w.WriteGlucoseReads(reads)
	w.Flush()

	if state.total != 25 {
		t.Errorf("TestWriteOfDayGlucoseReadBatchesInSingleCall failed: got a total of %d but expected %d", state.total, 25)
	}

	if state.batchCount != 2 {
		t.Errorf("TestWriteOfDayGlucoseReadBatchesInSingleCall failed: got a batchCount of %d but expected %d", state.batchCount, 2)
	}

	if state.writeCount != 2 {
		t.Errorf("TestWriteOfDayGlucoseReadBatchesInSingleCall failed: got a writeCount of %d but expected %d", state.writeCount, 2)
	}
}

func TestWriteOfHourlyGlucoseReadBatch(t *testing.T) {
	state := NewGlucoseWriterState()
	w := NewGlucoseStreamerDuration(NewStatsGlucoseReadWriter(state), time.Hour*1)

	ct, _ := time.Parse("02/01/2006 15:04", "18/04/2014 00:00")

	for i := 0; i < 13; i++ {
		readTime := ct.Add(time.Duration(i*5) * time.Minute)
		w, _ = w.WriteGlucoseRead(apimodel.GlucoseRead{apimodel.Time{apimodel.GetTimeMillis(readTime), "America/Montreal"}, apimodel.MG_PER_DL, float32(i)})
	}

	t.Logf("state is %p: %v", state, state)
	if state.total != 12 {
		t.Errorf("TestWriteOfHourlyGlucoseReadBatch failed: got a total of %d but expected %d", state.total, 12)
	}

	if state.batchCount != 1 {
		t.Errorf("TestWriteOfHourlyGlucoseReadBatch failed: got a batchCount of %d but expected %d", state.batchCount, 1)
	}

	if state.writeCount != 1 {
		t.Errorf("TestWriteOfHourlyGlucoseReadBatch failed: got a writeCount of %d but expected %d", state.writeCount, 1)
	}

	// Flushing should trigger the trailing read to be written
	w, _ = w.Flush()

	t.Logf("state is %p: %v", state, state)
	if state.total != 13 {
		t.Errorf("TestWriteOfHourlyGlucoseReadBatch failed: got a total of %d but expected %d", state.total, 13)
	}

	if state.batchCount != 2 {
		t.Errorf("TestWriteOfHourlyGlucoseReadBatch failed: got a batchCount of %d but expected %d", state.batchCount, 2)
	}

	if state.writeCount != 2 {
		t.Errorf("TestWriteOfHourlyGlucoseReadBatch failed: got a writeCount of %d but expected %d", state.writeCount, 2)
	}
}

func TestWriteOfMultipleGlucoseReadBatches(t *testing.T) {
	state := NewGlucoseWriterState()
	w := NewGlucoseStreamerDuration(NewStatsGlucoseReadWriter(state), time.Hour*1)

	ct, _ := time.Parse("02/01/2006 15:04", "18/04/2014 00:00")

	for i := 0; i < 25; i++ {
		readTime := ct.Add(time.Duration(i*5) * time.Minute)
		w, _ = w.WriteGlucoseRead(apimodel.GlucoseRead{apimodel.Time{apimodel.GetTimeMillis(readTime), "America/Montreal"}, apimodel.MG_PER_DL, float32(i)})
	}

	if state.total != 24 {
		t.Errorf("TestWriteOfMultipleGlucoseReadBatches failed: got a total of %d but expected %d", state.total, 24)
	}

	if state.batchCount != 2 {
		t.Errorf("TestWriteOfMultipleGlucoseReadBatches failed: got a batchCount of %d but expected %d", state.batchCount, 2)
	}

	if state.writeCount != 2 {
		t.Errorf("TestWriteOfMultipleGlucoseReadBatches failed: got a writeCount of %d but expected %d", state.writeCount, 2)
	}

	// Flushing should trigger the trailing read to be written
	w, _ = w.Flush()

	if state.total != 25 {
		t.Errorf("TestWriteOfMultipleGlucoseReadBatches failed: got a total of %d but expected %d", state.total, 13)
	}

	if state.batchCount != 3 {
		t.Errorf("TestWriteOfMultipleGlucoseReadBatches failed: got a batchCount of %d but expected %d", state.batchCount, 3)
	}

	if state.writeCount != 3 {
		t.Errorf("TestWriteOfMultipleGlucoseReadBatches failed: got a writeCount of %d but expected %d", state.writeCount, 3)
	}
}

func TestGlucoseStreamerWithBufferedIO(t *testing.T) {
	state := NewGlucoseWriterState()
	bufferedWriter := bufio.NewGlucoseReadWriterSize(NewStatsGlucoseReadWriter(state), 2)
	w := NewGlucoseStreamerDuration(bufferedWriter, apimodel.DAY_OF_DATA_DURATION)

	ct, _ := time.Parse("02/01/2006 15:04", "18/04/2014 00:00")

	for b := 0; b < 3; b++ {
		for i := 0; i < 48; i++ {
			readTime := ct.Add(time.Duration(b*48+i) * 30 * time.Minute)
			w, _ = w.WriteGlucoseRead(apimodel.GlucoseRead{apimodel.Time{apimodel.GetTimeMillis(readTime), "America/Montreal"}, apimodel.MG_PER_DL, float32(b*48 + i)})
		}
	}

	w.Close()

	firstBatchTime, _ := time.Parse("02/01/2006 15:04", "18/04/2014 00:00")
	if _, ok := state.batches[firstBatchTime.Unix()]; !ok {
		t.Errorf("TestGlucoseStreamerWithBufferedIO test failed: could not find a batch starting with a read time of [%v]/ts[%d] in batches: [%v]", firstBatchTime, firstBatchTime.Unix(), state.batches)
	}

	secondBatchTime := firstBatchTime.Add(time.Duration(24) * time.Hour)
	if _, ok := state.batches[secondBatchTime.Unix()]; !ok {
		t.Errorf("TestGlucoseStreamerWithBufferedIO test failed: could not find a batch starting with a read time of [%v]/ts[%d] in batches: [%v]", secondBatchTime, secondBatchTime.Unix(), state.batches)
	}

	thirdBatchTime := firstBatchTime.Add(time.Duration(48) * time.Hour)
	if _, ok := state.batches[thirdBatchTime.Unix()]; !ok {
		t.Errorf("TestGlucoseStreamerWithBufferedIO test failed: could not find a batch starting with a read time of [%v]/ts[%d] in batches: [%v]", thirdBatchTime, thirdBatchTime.Unix(), state.batches)
	}
}

func TestGlucoseBatchBoundaries(t *testing.T) {
	state := NewGlucoseWriterState()
	bufferedWriter := bufio.NewGlucoseReadWriterSize(NewStatsGlucoseReadWriter(state), 2)
	w := NewGlucoseStreamerDuration(bufferedWriter, apimodel.DAY_OF_DATA_DURATION)

	ct, _ := time.Parse("02/01/2006 15:04", "18/04/2014 01:00")
	t.Logf("Start time of test is %v", ct)

	for b := 0; b < 3; b++ {
		for i := 0; i < 48; i++ {
			readTime := ct.Add(time.Duration(b*48+i) * 30 * time.Minute)
			w, _ = w.WriteGlucoseRead(apimodel.GlucoseRead{apimodel.Time{apimodel.GetTimeMillis(readTime), "America/Montreal"}, apimodel.MG_PER_DL, float32(b*48 + i)})
		}
	}

	w.Close()

	// Fist batch still starts with the first read which isn't a day boundary because we're just keeping track of an array of reads and
	// therefore will have the first read potentially not line up with the data
	firstBatchTime, _ := time.Parse("02/01/2006 15:04", "18/04/2014 01:00")
	if _, ok := state.batches[firstBatchTime.Unix()]; !ok {
		t.Errorf("TestGlucoseBatchBoundaries test failed: could not find first batch starting with a read time of [%v]/ts[%d] in batches: [%v]", firstBatchTime, firstBatchTime.Unix(), state.batches)
	}

	// Second batch starts at the truncated day boundary because we have a matching read that starts with it
	secondBatchTime, _ := time.Parse("02/01/2006 15:04", "19/04/2014 00:00")
	if _, ok := state.batches[secondBatchTime.Unix()]; !ok {
		t.Errorf("TestGlucoseBatchBoundaries test failed: could not find second batch starting with a read time of [%v]/ts[%d] in batches: [%v]", secondBatchTime, secondBatchTime.Unix(), state.batches)
	}

	// Third batch starts at the truncated day boundary because we have a matching read that starts with it
	thirdBatchTime, _ := time.Parse("02/01/2006 15:04", "20/04/2014 00:00")
	if _, ok := state.batches[thirdBatchTime.Unix()]; !ok {
		t.Errorf("TestGlucoseBatchBoundaries test failed: could not find third batch starting with a read time of [%v]/ts[%d] in batches: [%v]", thirdBatchTime, thirdBatchTime.Unix(), state.batches)
	}

	// Fourth batch starts at the truncated day boundary because we have a matching read that starts with it
	fourthBatchTime, _ := time.Parse("02/01/2006 15:04", "21/04/2014 00:00")
	if _, ok := state.batches[fourthBatchTime.Unix()]; !ok {
		t.Errorf("TestGlucoseBatchBoundaries test failed: could not find third batch starting with a read time of [%v]/ts[%d] in batches: [%v]", fourthBatchTime, fourthBatchTime.Unix(), state.batches)
	}
}

func BenchmarkStreamerWithBufferedIO(b *testing.B) {
	for n := 0; n < b.N; n++ {
		state := NewGlucoseWriterState()
		bufferedWriter := bufio.NewGlucoseReadWriterSize(NewStatsGlucoseReadWriter(state), 2)
		w := NewGlucoseStreamerDuration(bufferedWriter, apimodel.DAY_OF_DATA_DURATION)

		ct, _ := time.Parse("02/01/2006 00:15", "18/04/2014 00:00")

		for j := 0; j < 3; j++ {
			for i := 0; i < 288; i++ {
				readTime := ct.Add(time.Duration(j*288+i) * 5 * time.Minute)
				w, _ = w.WriteGlucoseRead(apimodel.GlucoseRead{apimodel.Time{apimodel.GetTimeMillis(readTime), "America/Montreal"}, apimodel.MG_PER_DL, float32(j*288 + i)})
			}
		}

		w.Close()
	}
}
