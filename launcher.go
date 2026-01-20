package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func main() {
	fmt.Println("Запуск GophKeeper...")

	clientName := "gophkeeper"
	if runtime.GOOS == "windows" {
		clientName = "gophkeeper.exe"
	}
	// запускаем севре на фоне
	server := exec.Command("go", "run", "./cmd/server/main.go")
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr

	if err := server.Start(); err != nil {
		fmt.Printf("Ошибка запуска сервера: %v\n", err)
		return
	}

	time.Sleep(3 * time.Second)
	// собираем клиента
	if _, err := os.Stat(clientName); os.IsNotExist(err) {
		fmt.Println("Сборка клиента...")
		build := exec.Command("go", "build", "-o", clientName, "./cmd/gophkeeper/main.go")
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		build.Run()
		// если не винда даём права
		if runtime.GOOS != "windows" {
			os.Chmod(clientName, 0755)
		}
	}

	fmt.Println("Сервер запущен")
	// пишем как запускать агента
	if runtime.GOOS == "windows" {
		fmt.Println("Данный терминал не закрывай. Открой новый и запускай: .\\gophkeeper.exe")
	} else {
		fmt.Println("Данный терминал не закрывай. Открой новый и запускай: ./gophkeeper")
	}

	server.Wait()
}
