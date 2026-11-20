# Сборка настольного приложения «Балансировщик команд».
#
# Windows:  mingw32-make          (или make, если установлен GNU make)
# Linux/macOS: make
#
# Нужны Go 1.24+ и C-компилятор (gcc / MinGW-w64): Fyne использует CGO.

APP      := Teams
BUILD    := build
DATA     := programmers.json

export CGO_ENABLED := 1

ifeq ($(OS),Windows_NT)
  SHELL    := cmd.exe
  EXE      := $(BUILD)/$(APP).exe
  # -H=windowsgui — без чёрного окна консоли при запуске
  LDFLAGS  := -s -w -H=windowsgui
  COPYDATA := copy /Y $(DATA) $(BUILD)\$(DATA) >NUL && xcopy /E /I /Y /Q examples $(BUILD)\examples >NUL
  CLEAN    := if exist $(BUILD) rmdir /S /Q $(BUILD)
else
  EXE      := $(BUILD)/$(APP)
  LDFLAGS  := -s -w
  COPYDATA := cp $(DATA) $(BUILD)/$(DATA) && cp -r examples $(BUILD)/
  CLEAN    := rm -rf $(BUILD)
endif

.PHONY: all build run test vet clean help

all: build

## build — собрать приложение в build/ (рядом кладутся programmers.json и examples/)
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(EXE) .
	$(COPYDATA)
	@echo Готово: $(EXE)

## run — собрать и запустить
run: build
	cd $(BUILD) && $(if $(filter Windows_NT,$(OS)),$(APP).exe,./$(APP))

## test — запустить тесты алгоритма
test:
	go test ./...

## vet — статический анализ
vet:
	go vet ./...

## clean — удалить build/
clean:
	$(CLEAN)

help:
	@echo Цели: build (по умолчанию), run, test, vet, clean
