package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/arran4/golang-ical"
	"github.com/google/uuid"
)

type Course struct {
	title       string
	id          string
	room        string
	startHour   int
	startMinute int
	endHour     int
	endMinute   int
}

func parseTime(s string) int {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	return int(v)
}

func setHourMin(t time.Time, hour int, minute int) time.Time {
	result := time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		hour, minute, 0, 0, t.Location())
	return result
}

func getWeekDay(t time.Time) string {
	return strings.ToUpper(t.Weekday().String()[:2])
}

func findStartDate(ref time.Time, weekdays []string) time.Time {
	weekdayMap := map[string]int{
		"MO": 0,
		"TU": 1,
		"WE": 2,
		"TH": 3,
		"FR": 4,
		"SA": 5,
		"SU": 6,
	}
	curr := weekdayMap[getWeekDay(ref)]
	for _, d := range weekdays {
		diff := weekdayMap[d] - curr
		if diff >= 0 {
			return ref.AddDate(0, 0, diff)
		}
	}
	panic("something wrong")
}

const (
	icalTimestampFormatLocal = "20060102T150405"
	inputDateFormat          = "02/01/2006"
)

func setStartAt(event *ics.VEvent, t time.Time) {
	event.SetProperty(ics.ComponentPropertyDtStart, t.Format(icalTimestampFormatLocal))
}

func setEndAt(event *ics.VEvent, t time.Time) {
	event.SetProperty(ics.ComponentPropertyDtEnd, t.Format(icalTimestampFormatLocal))
}

func parseInputDate(s string) time.Time {
	result, err := time.Parse(inputDateFormat, s)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

func createEvent(course Course, startTime time.Time, endTime time.Time, weekdays []string) *ics.VEvent {
	event := ics.NewEvent(uuid.New().String())
	event.SetSummary(fmt.Sprintf("%s, %s", course.title, course.id))
	event.SetLocation(course.room)
	event.SetDtStampTime(time.Now())

	d := findStartDate(startTime, weekdays)
	start := setHourMin(d, course.startHour, course.startMinute)
	setStartAt(event, start)
	end := setHourMin(d, course.endHour, course.endMinute)
	setEndAt(event, end)

	byday := strings.Join(weekdays[:], ",")
	until := endTime.Format(icalTimestampFormatLocal)
	event.SetProperty(
		ics.ComponentPropertyRrule,
		fmt.Sprintf("FREQ=WEEKLY;WKST=SU;BYDAY=%s;UNTIL=%s", byday, until))
  
  return event
}

func main() {
  file, err := os.Open(os.Args[1])
  if err != nil {
    log.Fatal(err)
  }
  defer file.Close()

  var semesterStart time.Time
  var midtermStart time.Time
  var midtermEnd time.Time
  var semesterEnd time.Time

  scanner := bufio.NewScanner(file)
  for i := 0; i < 4; i++ {
    if scanner.Scan() {
      if i == 0 {
        semesterStart = parseInputDate(scanner.Text())
      }
      if i == 1 {
        midtermStart = parseInputDate(scanner.Text())
        midtermStart = midtermStart.AddDate(0, 0, -1) // last day of class before midterm 
      }
      if i == 2 {
        midtermEnd = parseInputDate(scanner.Text())
        midtermEnd = midtermEnd.AddDate(0, 0, 1) // first day of class after midterm
      }
      if i == 3 {
        semesterEnd = parseInputDate(scanner.Text())
      }
    }
  }
  
  var table string
  for scanner.Scan() {
    table += scanner.Text() + "\n"
  }

	weekdayMap := map[int]string{
		1: "MO",
		2: "TU",
		3: "WE",
		4: "TH",
		5: "FR",
		6: "SA",
		7: "SU",
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(table))

	if err != nil {
		log.Fatal(err)
	}

	courses := make(map[Course][]string)

	doc.Find(".day-time-cell").Each(func(i int, s *goquery.Selection) {
		s.Find(".text-truncate").Each(func(j int, nodes *goquery.Selection) {
			var data Course

			nodes.ChildrenFiltered("span").Each(func(k int, node *goquery.Selection) {
				trimmedString := strings.TrimSpace(node.Text())
				if k == 0 {
					data.title = trimmedString
				}
				if k == 1 {
					data.id = trimmedString
				}
				if k == 2 {
					splited := strings.Split(trimmedString, ",")
					data.room = strings.TrimSpace(splited[0])
					t := strings.TrimSpace(splited[1])[:11]

					data.startHour = parseTime(t[:2])
					data.startMinute = parseTime(t[2:4])
					data.endHour = parseTime(t[7:9])
					data.endMinute = parseTime(t[9:11])
				}
			})
			courses[data] = append(courses[data], weekdayMap[i])
		})
	})

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetCalscale("GREGORIAN")
	cal.SetTimezoneId("Asia/Bangkok")

	for course, weekdays := range courses {
    e1 := createEvent(course, semesterStart, midtermStart, weekdays)
    e2 := createEvent(course, midtermEnd, semesterEnd, weekdays)
    cal.AddVEvent(e1)
    cal.AddVEvent(e2)
	}


  outFile, err := os.Create("cal.ics")
  if err != nil {
    log.Fatal(err)
  }
  defer outFile.Close()

  _, err = outFile.WriteString(cal.Serialize())
  if err != nil {
    log.Fatal(err)
  }
}
