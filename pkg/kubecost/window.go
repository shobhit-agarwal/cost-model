package kubecost

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/kubecost/cost-model/pkg/util/timeutil"

	"github.com/kubecost/cost-model/pkg/env"
	"github.com/kubecost/cost-model/pkg/thanos"
)

const (
	minutesPerDay  = 60 * 24
	minutesPerHour = 60
	hoursPerDay    = 24
)

// RoundBack rounds the given time back to a multiple of the given resolution
// in the given time's timezone.
// e.g. 2020-01-01T12:37:48-0700, 24h = 2020-01-01T00:00:00-0700
func RoundBack(t time.Time, resolution time.Duration) time.Time {
	_, offSec := t.Zone()
	return t.Add(time.Duration(offSec) * time.Second).Truncate(resolution).Add(-time.Duration(offSec) * time.Second)
}

// RoundForward rounds the given time forward to a multiple of the given resolution
// in the given time's timezone.
// e.g. 2020-01-01T12:37:48-0700, 24h = 2020-01-02T00:00:00-0700
func RoundForward(t time.Time, resolution time.Duration) time.Time {
	back := RoundBack(t, resolution)
	if back.Equal(t) {
		// The given time is exactly a multiple of the given resolution
		return t
	}
	return back.Add(resolution)
}

// Window defines a period of time with a start and an end. If either start or
// end are nil it indicates an open time period.
type Window struct {
	start *time.Time
	end   *time.Time
}

// NewWindow creates and returns a new Window instance from the given times
func NewWindow(start, end *time.Time) Window {
	return Window{
		start: start,
		end:   end,
	}
}

// NewClosedWindow creates and returns a new Window instance from the given
// times, which cannot be nil, so they are value types.
func NewClosedWindow(start, end time.Time) Window {
	return Window{
		start: &start,
		end:   &end,
	}
}

// ParseWindowUTC attempts to parse the given string into a valid Window. It
// accepts several formats, returning an error if the given string does not
// match one of the following:
// - named intervals: "today", "yesterday", "week", "month", "lastweek", "lastmonth"
// - durations: "24h", "7d", etc.
// - date ranges: "2020-04-01T00:00:00Z,2020-04-03T00:00:00Z", etc.
// - timestamp ranges: "1586822400,1586908800", etc.
func ParseWindowUTC(window string) (Window, error) {
	return parseWindow(window, time.Now().UTC())
}

// ParseWindowWithOffsetString parses the given window string within the context of
// the timezone defined by the UTC offset string of format -07:00, +01:30, etc.
func ParseWindowWithOffsetString(window string, offset string) (Window, error) {
	if offset == "UTC" || offset == "" {
		return ParseWindowUTC(window)
	}

	regex := regexp.MustCompile(`^(\+|-)(\d\d):(\d\d)$`)
	match := regex.FindStringSubmatch(offset)
	if match == nil {
		return Window{}, fmt.Errorf("illegal UTC offset: '%s'; should be of form '-07:00'", offset)
	}

	sig := 1
	if match[1] == "-" {
		sig = -1
	}

	hrs64, _ := strconv.ParseInt(match[2], 10, 64)
	hrs := sig * int(hrs64)

	mins64, _ := strconv.ParseInt(match[3], 10, 64)
	mins := sig * int(mins64)

	loc := time.FixedZone(fmt.Sprintf("UTC%s", offset), (hrs*60*60)+(mins*60))
	now := time.Now().In(loc)
	return parseWindow(window, now)
}

// ParseWindowWithOffset parses the given window string within the context of
// the timezone defined by the UTC offset.
func ParseWindowWithOffset(window string, offset time.Duration) (Window, error) {
	loc := time.FixedZone("", int(offset.Seconds()))
	now := time.Now().In(loc)
	return parseWindow(window, now)
}

// parseWindow generalizes the parsing of window strings, relative to a given
// moment in time, defined as "now".
func parseWindow(window string, now time.Time) (Window, error) {
	// compute UTC offset in terms of minutes
	offHr := now.UTC().Hour() - now.Hour()
	offMin := (now.UTC().Minute() - now.Minute()) + (offHr * 60)
	offset := time.Duration(offMin) * time.Minute

	if window == "today" {
		start := now
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)

		end := start.Add(time.Hour * 24)

		return NewWindow(&start, &end), nil
	}

	if window == "yesterday" {
		start := now
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		start = start.Add(time.Hour * -24)

		end := start.Add(time.Hour * 24)

		return NewWindow(&start, &end), nil
	}

	if window == "week" {
		// now
		start := now
		// 00:00 today, accounting for timezone offset
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		// 00:00 Sunday of the current week
		start = start.Add(-24 * time.Hour * time.Duration(start.Weekday()))

		end := now

		return NewWindow(&start, &end), nil
	}

	if window == "lastweek" {
		// now
		start := now
		// 00:00 today, accounting for timezone offset
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		// 00:00 Sunday of last week
		start = start.Add(-24 * time.Hour * time.Duration(start.Weekday()+7))

		end := start.Add(7 * 24 * time.Hour)

		return NewWindow(&start, &end), nil
	}

	if window == "month" {
		// now
		start := now
		// 00:00 today, accounting for timezone offset
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		// 00:00 1st of this month
		start = start.Add(-24 * time.Hour * time.Duration(start.Day()-1))

		end := now

		return NewWindow(&start, &end), nil
	}

	if window == "month" {
		// now
		start := now
		// 00:00 today, accounting for timezone offset
		start = start.Truncate(time.Hour * 24)
		start = start.Add(offset)
		// 00:00 1st of this month
		start = start.Add(-24 * time.Hour * time.Duration(start.Day()-1))

		end := now

		return NewWindow(&start, &end), nil
	}

	if window == "lastmonth" {
		// now
		end := now
		// 00:00 today, accounting for timezone offset
		end = end.Truncate(time.Hour * 24)
		end = end.Add(offset)
		// 00:00 1st of this month
		end = end.Add(-24 * time.Hour * time.Duration(end.Day()-1))

		// 00:00 last day of last month
		start := end.Add(-24 * time.Hour)
		// 00:00 1st of last month
		start = start.Add(-24 * time.Hour * time.Duration(start.Day()-1))

		return NewWindow(&start, &end), nil
	}

	// Match duration strings; e.g. "45m", "24h", "7d"
	regex := regexp.MustCompile(`^(\d+)(m|h|d)$`)
	match := regex.FindStringSubmatch(window)
	if match != nil {
		dur := time.Minute
		if match[2] == "h" {
			dur = time.Hour
		}
		if match[2] == "d" {
			dur = 24 * time.Hour
		}

		num, _ := strconv.ParseInt(match[1], 10, 64)

		end := now
		start := end.Add(-time.Duration(num) * dur)

		return NewWindow(&start, &end), nil
	}

	// Match duration strings with offset; e.g. "45m offset 15m", etc.
	regex = regexp.MustCompile(`^(\d+)(m|h|d) offset (\d+)(m|h|d)$`)
	match = regex.FindStringSubmatch(window)
	if match != nil {
		end := now

		offUnit := time.Minute
		if match[4] == "h" {
			offUnit = time.Hour
		}
		if match[4] == "d" {
			offUnit = 24 * time.Hour
		}

		offNum, _ := strconv.ParseInt(match[3], 10, 64)

		end = end.Add(-time.Duration(offNum) * offUnit)

		durUnit := time.Minute
		if match[2] == "h" {
			durUnit = time.Hour
		}
		if match[2] == "d" {
			durUnit = 24 * time.Hour
		}

		durNum, _ := strconv.ParseInt(match[1], 10, 64)

		start := end.Add(-time.Duration(durNum) * durUnit)

		return NewWindow(&start, &end), nil
	}

	// Match timestamp pairs, e.g. "1586822400,1586908800" or "1586822400-1586908800"
	regex = regexp.MustCompile(`^(\d+)[,|-](\d+)$`)
	match = regex.FindStringSubmatch(window)
	if match != nil {
		s, _ := strconv.ParseInt(match[1], 10, 64)
		e, _ := strconv.ParseInt(match[2], 10, 64)
		start := time.Unix(s, 0)
		end := time.Unix(e, 0)
		return NewWindow(&start, &end), nil
	}

	// Match RFC3339 pairs, e.g. "2020-04-01T00:00:00Z,2020-04-03T00:00:00Z"
	rfc3339 := `\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\dZ`
	regex = regexp.MustCompile(fmt.Sprintf(`(%s),(%s)`, rfc3339, rfc3339))
	match = regex.FindStringSubmatch(window)
	if match != nil {
		start, _ := time.Parse(time.RFC3339, match[1])
		end, _ := time.Parse(time.RFC3339, match[2])
		return NewWindow(&start, &end), nil
	}

	return Window{nil, nil}, fmt.Errorf("illegal window: %s", window)
}

// ApproximatelyEqual returns true if the start and end times of the two windows,
// respectively, are within the given threshold of each other.
func (w Window) ApproximatelyEqual(that Window, threshold time.Duration) bool {
	return approxEqual(w.start, that.start, threshold) && approxEqual(w.end, that.end, threshold)
}

func approxEqual(x *time.Time, y *time.Time, threshold time.Duration) bool {
	// both times are nil, so they are equal
	if x == nil && y == nil {
		return true
	}

	// one time is nil, but the other is not, so they are not equal
	if x == nil || y == nil {
		return false
	}

	// neither time is nil, so they are approximately close if their times are
	// within the given threshold
	delta := math.Abs((*x).Sub(*y).Seconds())
	return delta < threshold.Seconds()
}

func (w Window) Clone() Window {
	var start, end *time.Time
	var s, e time.Time

	if w.start != nil {
		s = *w.start
		start = &s
	}

	if w.end != nil {
		e = *w.end
		end = &e
	}

	return NewWindow(start, end)
}

func (w Window) Contains(t time.Time) bool {
	if w.start != nil && t.Before(*w.start) {
		return false
	}

	if w.end != nil && t.After(*w.end) {
		return false
	}

	return true
}

func (w Window) ContainsWindow(that Window) bool {
	// only support containing closed windows for now
	// could check if openness is compatible with closure
	if that.IsOpen() {
		return false
	}

	return w.Contains(*that.start) && w.Contains(*that.end)
}

func (w Window) Duration() time.Duration {
	if w.IsOpen() {
		// TODO test
		return time.Duration(math.Inf(1.0))
	}

	return w.end.Sub(*w.start)
}

func (w Window) End() *time.Time {
	return w.end
}

func (w Window) Equal(that Window) bool {
	if w.start != nil && that.start != nil && !w.start.Equal(*that.start) {
		// starts are not nil, but not equal
		return false
	}

	if w.end != nil && that.end != nil && !w.end.Equal(*that.end) {
		// ends are not nil, but not equal
		return false
	}

	if (w.start == nil && that.start != nil) || (w.start != nil && that.start == nil) {
		// one start is nil, the other is not
		return false
	}

	if (w.end == nil && that.end != nil) || (w.end != nil && that.end == nil) {
		// one end is nil, the other is not
		return false
	}

	// either both starts are nil, or they match; likewise for the ends
	return true
}

func (w Window) ExpandStart(start time.Time) Window {
	if w.start == nil || start.Before(*w.start) {
		w.start = &start
	}
	return w
}

func (w Window) ExpandEnd(end time.Time) Window {
	if w.end == nil || end.After(*w.end) {
		w.end = &end
	}
	return w
}

func (w Window) Expand(that Window) Window {
	if that.start == nil {
		w.start = nil
	} else {
		w = w.ExpandStart(*that.start)
	}

	if that.end == nil {
		w.end = nil
	} else {
		w = w.ExpandEnd(*that.end)
	}

	return w
}

func (w Window) ContractStart(start time.Time) Window {
	if w.start == nil || start.After(*w.start) {
		w.start = &start
	}
	return w
}

func (w Window) ContractEnd(end time.Time) Window {
	if w.end == nil || end.Before(*w.end) {
		w.end = &end
	}
	return w
}

func (w Window) Contract(that Window) Window {
	if that.start != nil {
		w = w.ContractStart(*that.start)
	}

	if that.end != nil {
		w = w.ContractEnd(*that.end)
	}

	return w
}

func (w Window) Hours() float64 {
	if w.IsOpen() {
		return math.Inf(1)
	}

	return w.end.Sub(*w.start).Hours()
}

func (w Window) IsEmpty() bool {
	return !w.IsOpen() && w.end.Equal(*w.Start())
}

func (w Window) IsNegative() bool {
	return !w.IsOpen() && w.end.Before(*w.Start())
}

func (w Window) IsOpen() bool {
	return w.start == nil || w.end == nil
}

// TODO:CLEANUP make this unmarshalable (make Start and End public)
func (w Window) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	if w.start != nil {
		buffer.WriteString(fmt.Sprintf("\"start\":\"%s\",", w.start.Format(time.RFC3339)))
	} else {
		buffer.WriteString(fmt.Sprintf("\"start\":\"%s\",", "null"))
	}
	if w.end != nil {
		buffer.WriteString(fmt.Sprintf("\"end\":\"%s\"", w.end.Format(time.RFC3339)))
	} else {
		buffer.WriteString(fmt.Sprintf("\"end\":\"%s\"", "null"))
	}
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func (w Window) Minutes() float64 {
	if w.IsOpen() {
		return math.Inf(1)
	}

	return w.end.Sub(*w.start).Minutes()
}

// Overlaps returns true iff the two given Windows share an amount of temporal
// coverage.
// TODO complete (with unit tests!) and then implement in AllocationSet.accumulate
// TODO:CLEANUP
// func (w Window) Overlaps(x Window) bool {
// 	if (w.start == nil && w.end == nil) || (x.start == nil && x.end == nil) {
// 		// one window is completely open, so overlap is guaranteed
// 		// <---------->
// 		//   ?------?
// 		return true
// 	}

// 	// Neither window is completely open (nil, nil), but one or the other might
// 	// still be future- or past-open.

// 	if w.start == nil {
// 		// w is past-open, future-closed
// 		// <------]

// 		if x.start != nil && !x.start.Before(*w.end) {
// 			// x starts after w ends (or eq)
// 			// <------]
// 			//          [------?
// 			return false
// 		}

// 		// <-----]
// 		//    ?-----?
// 		return true
// 	}

// 	if w.end == nil {
// 		// w is future-open, past-closed
// 		// [------>

// 		if x.end != nil && !x.end.After(*w.end) {
// 			// x ends before w begins (or eq)
// 			//          [------>
// 			// ?------]
// 			return false
// 		}

// 		//    [------>
// 		// ?------?
// 		return true
// 	}

// 	// Now we know w is closed, but we don't know about x
// 	//  [------]
// 	//     ?------?
// 	if x.start == nil {
// 		// TODO
// 	}

// 	if x.end == nil {
// 		// TODO
// 	}

// 	// Both are closed.

// 	if !x.start.Before(*w.end) && !x.end.Before(*w.end) {
// 		// x starts and ends after w ends
// 		// [------]
// 		//          [------]
// 		return false
// 	}

// 	if !x.start.After(*w.start) && !x.end.After(*w.start) {
// 		// x starts and ends before w starts
// 		//          [------]
// 		// [------]
// 		return false
// 	}

// 	// w and x must overlap
// 	//    [------]
// 	// [------]
// 	return true
// }

func (w Window) Set(start, end *time.Time) {
	w.start = start
	w.end = end
}

// Shift adds the given duration to both the start and end times of the window
func (w Window) Shift(dur time.Duration) Window {
	if w.start != nil {
		s := w.start.Add(dur)
		w.start = &s
	}

	if w.end != nil {
		e := w.end.Add(dur)
		w.end = &e
	}

	return w
}

func (w Window) Start() *time.Time {
	return w.start
}

func (w Window) String() string {
	if w.start == nil && w.end == nil {
		return "[nil, nil)"
	}
	if w.start == nil {
		return fmt.Sprintf("[nil, %s)", w.end.Format("2006-01-02T15:04:05-0700"))
	}
	if w.end == nil {
		return fmt.Sprintf("[%s, nil)", w.start.Format("2006-01-02T15:04:05-0700"))
	}
	return fmt.Sprintf("[%s, %s)", w.start.Format("2006-01-02T15:04:05-0700"), w.end.Format("2006-01-02T15:04:05-0700"))
}

// DurationOffset returns durations representing the duration and offset of the
// given window
func (w Window) DurationOffset() (time.Duration, time.Duration, error) {
	if w.IsOpen() || w.IsNegative() {
		return 0, 0, fmt.Errorf("illegal window: %s", w)
	}

	duration := w.Duration()
	offset := time.Since(*w.End())

	return duration, offset, nil
}

// DurationOffsetForPrometheus returns strings representing durations for the
// duration and offset of the given window, factoring in the Thanos offset if
// necessary. Whereas duration is a simple duration string (e.g. "1d"), the
// offset includes the word "offset" (e.g. " offset 2d") so that the values
// returned can be used directly in the formatting string "some_metric[%s]%s"
// to generate the query "some_metric[1d] offset 2d".
func (w Window) DurationOffsetForPrometheus() (string, string, error) {
	duration, offset, err := w.DurationOffset()
	if err != nil {
		return "", "", err
	}

	// If using Thanos, increase offset to 3 hours, reducing the duration by
	// equal measure to maintain the same starting point.
	thanosDur := thanos.OffsetDuration()
	if offset < thanosDur && env.IsThanosEnabled() {
		diff := thanosDur - offset
		offset += diff
		duration -= diff
	}

	// If duration < 0, return an error
	if duration < 0 {
		return "", "", fmt.Errorf("negative duration: %s", duration)
	}

	// Negative offset means that the end time is in the future. Prometheus
	// fails for non-positive offset values, so shrink the duration and
	// remove the offset altogether.
	if offset < 0 {
		duration = duration + offset
		offset = 0
	}

	durStr, offStr := timeutil.DurationOffsetStrings(duration, offset)
	if offset < time.Minute {
		offStr = ""
	} else {
		offStr = " offset " + offStr
	}

	return durStr, offStr, nil
}

// DurationOffsetStrings returns formatted, Prometheus-compatible strings representing
// the duration and offset of the window in terms of days, hours, minutes, or seconds;
// e.g. ("7d", "1441m", "30m", "1s", "")
func (w Window) DurationOffsetStrings() (string, string) {
	dur, off, err := w.DurationOffset()
	if err != nil {
		return "", ""
	}

	return timeutil.DurationOffsetStrings(dur, off)
}

type BoundaryError struct {
	Requested Window
	Supported Window
	Message   string
}

func NewBoundaryError(req, sup Window, msg string) *BoundaryError {
	return &BoundaryError{
		Requested: req,
		Supported: sup,
		Message:   msg,
	}
}

func (be *BoundaryError) Error() string {
	if be == nil {
		return "<nil>"
	}

	return fmt.Sprintf("boundary error: requested %s; supported %s: %s", be.Requested, be.Supported, be.Message)
}
