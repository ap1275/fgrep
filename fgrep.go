package main

import (
  "os"
  "regexp"
  "bufio"
  "io/ioutil"
  "path/filepath"
  "flag"
  "fmt"
  "sync"
  "runtime"
)

type pat []string

func (p *pat) String() string {
  return fmt.Sprintf("%s", *p)
}

func (p *pat) Set(v string) error {
  *p = append(*p, v)
  return nil
}

var(
  buffer_size *int
  show_only_file_status *bool
  pattern pat
  only_find_file bool
  absolute_path *bool
  num_parallel *int
)

func walk(dir string) ([]string, error) {
  files, err := ioutil.ReadDir(dir)
  var path []string
  if err != nil {
    return path, err
  }
  for _, f := range files {
    if f.IsDir() {
      p, e := walk(filepath.Join(dir, f.Name()))
      if e == nil {
        path = append(path, p...)
        continue
      } else {
        return p, e
      }
    }
    if pattern == nil {
      path = append(path, filepath.Join(dir, f.Name()))
    } else {
      for _, m := range pattern {
        r := regexp.MustCompile(m)
        if r.Match([]byte(f.Name())) {
          path = append(path, filepath.Join(dir, f.Name()))
        }
      }
    }
  }
  return path, err
}

func exec(f string, wg *sync.WaitGroup, r *regexp.Regexp, sph chan int) {
  defer wg.Done()
  sph <- 1
  p, e := filepath.Abs(f)
  if !*absolute_path {
    p = f
  }
  if e != nil {
    panic(e)
  }
  if r == nil {
    fmt.Println(p)
    return
  }
  fp, err := os.Open(f)
  if err != nil {
    fmt.Println(err)
    return
  }
  defer fp.Close()
  b := bufio.NewReaderSize(fp, *buffer_size)
  var i int
  var l []byte
  for ; err == nil; l, err = b.ReadBytes('\n') {
    if r != nil && r.Match(l) {
      if *show_only_file_status {
        fmt.Printf("%s:%d\n", p, i)
      } else {
        fmt.Printf("[%s:%d]%s", p, i, l)
      }
    }
    i++
  }
  <- sph
}

func search(files []string, r *regexp.Regexp) {
  w := new(sync.WaitGroup)
  sph := make(chan int, *num_parallel)
  for _, file := range files {
    w.Add(1)
    go exec(file, w, r, sph)
  }
  w.Wait()
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())
  only_find_file = false
  buffer_size = flag.Int("b", 4096, "buffer size of each lines")
  show_only_file_status = flag.Bool("s", false, "show only file status(this will be ignored if set -f and not set -r)")
  num_parallel = flag.Int("n", 1000, "max number which able to run as parallel")
  absolute_path = flag.Bool("a", false, "show path as absolute. default is relative")
  dir := flag.String("p", ".", "root path to start searching")
  r := flag.String("r", "", "regex for each lines")
  flag.Var(&pattern, "f", "regex for files pattern. this flag can be multiple. all files will match if this flag is null")
  flag.Parse()
  if *r == "" && pattern != nil {
    *show_only_file_status = true
    only_find_file = true
  }
  if !only_find_file && *r == "" {
    fmt.Println("set -r flag")
    return
  }
  p, e := walk(*dir)
  if e != nil {
    fmt.Println(e)
    return
  }
  if !only_find_file {
    reg := regexp.MustCompile(*r)
    search(p, reg)
  } else {
    search(p, nil)
  }
}
