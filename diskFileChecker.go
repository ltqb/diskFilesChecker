package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

//获取目录dir下的文件大小
func walkDir(dir string, wg *sync.WaitGroup, fileSize chan int64) {
	defer wg.Done()
	for _, entry := range dirents(dir) {
		if entry.IsDir() { //目录
			wg.Add(1)
			subDir := filepath.Join(dir, entry.Name())
			go walkDir(subDir, wg, fileSize)
		} else {
			accessSeconds := entry.Sys().(*syscall.Stat_t).Atim.Nano()
			accessTime := time.Unix(accessSeconds/1e9, 0)
			//fmt.Println(accessTime)
			//fmt.Print(accessTime)
			cTime := time.Time.AddDate(time.Now(), 0, -1, 0)
			if accessTime.Before(cTime) {
				//subDir := filepath.Join(dir, entry.Name())
				//fmt.Println(subDir)
				fileSize <- entry.Size()
			}
		}
	}
}

//sema is a counting semaphore for limiting concurrency in dirents
var sema = make(chan struct{}, 48)

//读取目录dir下的文件信息
func dirents(dir string) []os.FileInfo {
	sema <- struct{}{}
	defer func() { <-sema }()
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "du: %v\n", err)
		return nil
	}
	return entries
}

//输出文件数量的大小
func printDiskUsage(nfiles, nbytes int64) {
	fmt.Printf("%d files %.6f GB\n", nfiles, float64(nbytes)/1e9)
}

//提供-v 参数会显示程序进度信息
var verbose = flag.Bool("v", false, "show verbose progress messages")

func Start() {
	flag.Parse()
	roots := flag.Args() //需要统计的目录
	if len(roots) == 0 {
		roots = []string{"."}
	}
	fileSizes := make(chan int64)
	var wg sync.WaitGroup
	for _, root := range roots {
		wg.Add(1)
		go walkDir(root, &wg, fileSizes)
	}
	go func() {
		wg.Wait() //等待goroutine结束
		close(fileSizes)
	}()
	var tick <-chan time.Time
	if *verbose {
		tick = time.Tick(100 * time.Millisecond) //输出时间间隔
	}
	var nfiles, nbytes int64
loop:
	for {
		select {
		case size, ok := <-fileSizes:
			if !ok {
				break loop
			}
			nfiles++
			nbytes += size
		case <-tick:
			printDiskUsage(nfiles, nbytes)
		}
	}
	printDiskUsage(nfiles, nbytes)
}

func main() {
	Start()
}
