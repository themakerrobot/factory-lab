// Factory Lab portable — 사이트 전체를 exe 하나에 담아 오프라인에서 실행한다.
//
//	사용법: FactoryLab.exe [-port 50040] [-no-open]
//
// 파이보 랩(pibo-lab)의 tools/portable 과 같은 구성이지만 로그인이 없다.
// 인증 API 를 호출하면 인터넷이 끊긴 교실에서 아예 못 들어가기 때문이다.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

//go:embed all:site
var siteFS embed.FS

const basePort = 50040 // 파이보 랩(50030)과 겹치지 않게

var (
	port   = flag.Int("port", basePort, "listen port (사용 중이면 다음 포트를 차례로 시도)")
	noOpen = flag.Bool("no-open", false, "브라우저 자동 열기 끄기")
)

// 로봇 연결(Web Serial)은 Chrome / Edge 에서만 된다. 기본 브라우저가 Firefox 면
// 3D 보기만 되므로, 크로미움 계열을 먼저 찾아 열고 없을 때만 기본 브라우저로 넘긴다.
func chromiumPaths() []string {
	switch runtime.GOOS {
	case "windows":
		var out []string
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LocalAppData")} {
			if base == "" {
				continue
			}
			out = append(out,
				filepath.Join(base, `Google\Chrome\Application\chrome.exe`),
				filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`))
		}
		return out
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"}
	default:
		return []string{"/usr/bin/google-chrome", "/usr/bin/chromium", "/usr/bin/chromium-browser", "/usr/bin/microsoft-edge"}
	}
}

// 찾으면 --app 모드(주소창 없는 창)로 띄운다. true = 크로미움으로 열었다.
func openBrowser(u string) bool {
	for _, p := range chromiumPaths() {
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			continue
		}
		if exec.Command(p, "--app="+u).Start() == nil {
			return true
		}
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	_ = cmd.Start()
	return false
}

func main() {
	flag.Parse()

	sub, err := fs.Sub(siteFS, "site")
	if err != nil {
		log.Fatal(err)
	}
	static := http.FileServer(http.FS(sub))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Go 의 기본 mime 표에 없어서 직접 지정한다
		if strings.HasSuffix(r.URL.Path, ".webmanifest") {
			w.Header().Set("Content-Type", "application/manifest+json")
		}
		// HTML 은 캐시에 눌러앉지 않게 (exe 를 새 버전으로 바꿨을 때 옛 화면이 뜨는 것 방지)
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/") || strings.HasSuffix(r.URL.Path, ".html") {
			w.Header().Set("Cache-Control", "no-cache")
		}
		static.ServeHTTP(w, r)
	})

	// localhost 에만 바인딩. 포트가 사용 중이면 다음 포트를 차례로 시도한다.
	p := *port
	var ln net.Listener
	for i := 0; i < 10; i++ {
		ln, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", p))
		if err == nil {
			break
		}
		ln = nil
		p++
	}
	if ln == nil {
		fmt.Println("빈 포트를 찾지 못했습니다:", *port)
		fmt.Scanln()
		return
	}

	addr := fmt.Sprintf("http://localhost:%d", p)
	fmt.Println("Factory Lab -", addr)
	if p != *port {
		// origin 이 바뀌면 브라우저에 저장된 영점 보정값·자세 프리셋이 따로 논다
		fmt.Printf("기본 포트 %d 가 사용 중이라 %d 로 열었습니다.\n", *port, p)
		fmt.Println("영점 보정값과 자세 프리셋은 포트마다 따로 저장되니 다시 맞춰야 할 수 있습니다.")
	}
	fmt.Println("이 창을 닫으면 종료됩니다.")

	if !*noOpen {
		go func() {
			time.Sleep(600 * time.Millisecond)
			if !openBrowser(addr) {
				fmt.Println("Chrome / Edge 를 찾지 못해 기본 브라우저로 열었습니다.")
				fmt.Println("로봇 연결은 Chrome 또는 Edge 에서만 됩니다. 다른 브라우저면 3D 보기만 가능합니다.")
			}
		}()
	}

	log.Fatal(http.Serve(ln, mux))
}
