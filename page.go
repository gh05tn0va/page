package page

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"sync"
)

type (
	Urls struct {
		data []string
		tag  string
	}

	PagingJob struct {
		BaseJob
		Tasks     []PagingTask
		Tags      map[string]string
		TaskAlias map[string]int
		Output    PagingOutput
	}

	Doc               = *goquery.Document
	OutputWithTag     []string
	OutputListWithTag []OutputWithTag
	PagingOutput      map[string]OutputListWithTag
	PagingTask        func(Doc) []OutputWithTag
)

/*
Output:
{
	"Tag 1": []OutputWithTag{

		OutputWithTag{

			"value1-1", // Task 1
			"value1-2", // Task 2
			...

		}, // Child 1

		OutputWithTag{

			"value2-1", // Task 1
			"value2-2", // Task 2
			...

		}, // Child 2

		...

	},

	"Tag 2": []OutputWithTag{...},

	...

}
*/

func OnOne(url string) *Urls {
	return OnMany([]string{url})
}

func OnMany(urls []string) *Urls {
	s := new(Urls)
	s.data = append(s.data, urls...)
	return s
}

func OnRange(format string, begin, end int) *Urls {
	s := new(Urls)
	for i := begin; i <= end; i++ {
		s.data = append(s.data, fmt.Sprintf(format, i))
	}
	return s
}

func (s *Urls) Tag(tag string) *Urls {
	s.tag = tag
	return s
}

func WorkerFunc(url string, j Job, lock *sync.Mutex) error {
	pj, ok := j.(*PagingJob)
	if !ok {
		log.Fatalf("Expected *page.PagingJob but got %T", j)
	}

	doc, err := GetPageBody(url)
	if err != nil {
		log.Printf("GET %s ERR %s", url, err.Error())
		return err
	}
	if DebugWorker {
		log.Printf("GET %s OK", url)
	}

	tag := pj.Tags[url]
	if tag == "" {
		tag = url
	}

	for i, taskFunc := range pj.Tasks {
		v := taskFunc(doc)
		if v != nil {
			lock.Lock()
			pj.Output[tag] = append(pj.Output[tag], v...)
			lock.Unlock()
		}
		if DebugWorker {
			log.Printf("TASK %d %s OK", i, url)
		}
	}

	return nil
}

func New() *PagingJob {
	pj := new(PagingJob)
	pj.Output = make(map[string]OutputListWithTag)
	pj.Tags = make(map[string]string)
	pj.TaskAlias = make(map[string]int)
	pj.WorkerFunc = WorkerFunc
	return pj
}

func (pj *PagingJob) AddRange(s *Urls) {
	pj.Set = append(pj.Set, s.data...)
	if s.tag != "" {
		for _, url := range s.data {
			pj.Tags[url] = s.tag
		}
	}
}

func (w *Worker) AddRange(format string, begin, end int) *Worker {
	s := new(Urls)
	for i := begin; i <= end; i++ {
		s.data = append(s.data, fmt.Sprintf(format, i))
	}
	return w.Add(s.data)
}

func (s *Urls) UseWorker(w *Worker) *Worker {
	w.Add(s.data)
	w.Job.(*PagingJob).AddRange(s)
	return w.Add(s.data)
}

func (s *Urls) AddTask(f PagingTask) *PagingJob {
	pj := New()
	pj.AddRange(s)
	return pj.AddTask(f)
}

func (pj *PagingJob) AddTask(f PagingTask) *PagingJob {
	pj.Tasks = append(pj.Tasks, f)
	return pj
}

func (s *Urls) Text() Job {
	pj := New()
	pj.AddRange(s)
	return pj.Text()
}

func (pj *PagingJob) Text() *PagingJob {
	pj.Tasks = append(pj.Tasks,
		func(doc Doc) []OutputWithTag {
			return []OutputWithTag{{doc.Text()}}
		})
	return pj
}

func (pj *PagingJob) Run() *Worker {
	w := new(Worker)
	w.Job = pj
	return w.Run()
}

func (pj *PagingJob) Get(s string) OutputListWithTag {
	return pj.Run().Out()[s]
}

func (w *Worker) Get(s string) OutputListWithTag {
	return w.Out()[s]
}

func (pj *PagingJob) Out() (out PagingOutput) {
	return pj.Run().Out()
}

func (w *Worker) Out() (out PagingOutput) {
	return w.Wait().Job.(*PagingJob).Output
}

func (o PagingOutput) Task(w *Worker, s string) map[string]OutputWithTag {
	i, ok := w.Job.(*PagingJob).TaskAlias[s]
	if !ok {
		return nil
	}
	return o.TaskN(i)
}

func (o OutputWithTag) Task(w *Worker, s string) string {
	i, ok := w.Job.(*PagingJob).TaskAlias[s]
	if !ok {
		return ""
	}
	return o.TaskN(i)
}

func (o OutputListWithTag) Task(w *Worker, s string) []string {
	i, ok := w.Job.(*PagingJob).TaskAlias[s]
	if !ok {
		return nil
	}
	return o.TaskN(i)
}

func (o PagingOutput) TaskN(i int) (out map[string]OutputWithTag) {
	out = make(map[string]OutputWithTag)
	for k, v := range o {
		// var v []OutputWithTag
		for _, vv := range v {
			out[k] = append(out[k], vv[i])
		}
	}
	return
}

func (o OutputWithTag) TaskN(i int) string {
	if len(o) <= i {
		return ""
	}
	return o[i]
}

func (o OutputListWithTag) TaskN(i int) (out []string) {
	for _, v := range o {
		vv := v.TaskN(i)
		if vv != "" {
			out = append(out, vv)
		}
	}
	return
}

func (o PagingOutput) List() (out OutputListWithTag) {
	for _, v := range o {
		// var v []OutputWithTag
		out = append(out, v...)
	}
	return
}

func (o PagingOutput) ListTask(w *Worker, s string) OutputWithTag {
	i, ok := w.Job.(*PagingJob).TaskAlias[s]
	if !ok {
		return nil
	}
	return o.ListTaskN(i)
}

func (o PagingOutput) ListTaskN(i int) (out OutputWithTag) {
	for _, v := range o {
		// var v []OutputWithTag
		for _, vv := range v {
			out = append(out, vv[i])
		}
	}
	return
}
