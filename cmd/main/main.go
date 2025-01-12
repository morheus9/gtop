package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/process"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func createBorderedFrame(inner tview.Primitive) *tview.Frame {
	frame := tview.NewFrame(inner).
		SetBorders(1, 1, 1, 1, 1, 1)
	frame.SetBorder(true).SetBorderColor(tcell.ColorLightSkyBlue)
	return frame
}

func main() {
	app := tview.NewApplication()
	logicalCpuCount, _ := cpu.Counts(true)
	cpuGauges := make([]*tvxwidgets.UtilModeGauge, logicalCpuCount)

	for i := 0; i < logicalCpuCount; i++ {
		cpuGauge := tvxwidgets.NewUtilModeGauge()
		cpuGauge.SetLabel(fmt.Sprintf(" CPU %d ", i+1))
		cpuGauge.SetLabelColor(tcell.ColorAntiqueWhite)
		cpuGauge.SetBorder(false)
		cpuGauges[i] = cpuGauge
		cpuGauge.SetRect(10, 3, 50, 3)

	}

	var wg sync.WaitGroup

	updateCpuGauges := func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered in updateCpuGauges: %v\n", r)
			}
		}()
		tick := time.NewTicker(1 * time.Second)
		defer tick.Stop()
		for {
			<-tick.C
			v, err := cpu.Percent(time.Second, true)
			if err != nil {
				fmt.Printf("Error getting CPU percent: %v\n", err)
				return
			}
			for i := 0; i < logicalCpuCount; i++ {
				cpuGauges[i].SetValue(v[i])
			}
			app.Draw()
		}
	}

	platform := tview.NewTextView()
	platform.SetLabel(" Platform       ")
	platform.SetBorder(false)
	platformOS, _, platformVersion, err := host.PlatformInformation()
	if err != nil {
		fmt.Println("Error retrieving platform information:", err)
		return
	}

	platform.SetText(fmt.Sprintf("%s %s", platformOS, platformVersion))

	loadAvg := tview.NewTextView()
	loadAvg.SetLabel(" Load average   ")
	loadAvg.SetBorder(false)
	loadAvg.SetRect(10, 3, 50, 3)

	uptime := tview.NewTextView()
	uptime.SetLabel(" Uptime         ")
	uptime.SetBorder(false)
	loadAvg.SetRect(10, 3, 50, 3)

	updateData := func() {
		// Получение информации о загрузке
		loadAvgInfo, err := load.Avg()
		if err != nil {
			loadAvg.SetText(fmt.Sprintf("Error retrieving load average: %v", err))
			return
		}
		loadAvg.SetText(fmt.Sprintf("1 min: %.2f\n5 min: %.2f\n15 min: %.2f",
			loadAvgInfo.Load1, loadAvgInfo.Load5, loadAvgInfo.Load15))
		loadAvg.SetText(fmt.Sprintf("%.2f    %.2f    %.2f",
			loadAvgInfo.Load1, loadAvgInfo.Load5, loadAvgInfo.Load15))

		// Получение информации о времени работы
		uptimeInfo, err := host.Uptime()
		if err != nil {
			uptime.SetText(fmt.Sprintf("Error retrieving uptime: %v", err))
			return
		}
		// Преобразование uptime в формат HH:MM:SS
		hours := uptimeInfo / 3600
		minutes := (uptimeInfo % 3600) / 60
		seconds := uptimeInfo % 60
		uptime.SetText(fmt.Sprintf("Uptime: %02d:%02d:%02d", hours, minutes, seconds))

	}

	go func() {
		for {
			updateData()
			time.Sleep(5 * time.Second)
		}
	}()
	memGauge := tvxwidgets.NewUtilModeGauge()
	memGauge.SetLabel(" mem   ")
	memGauge.SetRect(10, 3, 50, 3)
	memGauge.SetWarnPercentage(65)
	memGauge.SetCritPercentage(80)
	memGauge.SetBorder(false)

	updateMemGauge := func() {
		defer wg.Done()
		tick := time.NewTicker(1 * time.Second)
		defer tick.Stop()

		for range tick.C {
			v, err := mem.VirtualMemory()
			if err != nil {
				fmt.Printf("Error getting memory percent: %v\n", err)
				return
			}
			memoryUsed := float64(v.UsedPercent)
			memGauge.SetValue(memoryUsed)
			app.Draw()
		}
	}

	swapGauge := tvxwidgets.NewUtilModeGauge()
	swapGauge.SetLabel(" swap  ")
	swapGauge.SetRect(10, 3, 50, 3)
	swapGauge.SetWarnPercentage(65)
	swapGauge.SetCritPercentage(80)
	swapGauge.SetBorder(false)

	updateSwapGauge := func() {
		defer wg.Done()
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()

		for range tick.C {
			v, err := mem.VirtualMemory()
			if err != nil {
				fmt.Printf("Error getting swap memory: %v\n", err)
				return
			}
			swapUsed := float64(v.SwapCached)
			swapGauge.SetValue(swapUsed)
			app.Draw()
		}
	}

	cpuFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	for _, gauge := range cpuGauges {
		cpuFlex.AddItem(gauge, 1, 1, false)
	}

	processTable := tview.NewTable().SetBorders(true)
	processTable.SetFixed(1, 0)

	sortBy := "cpu"
	updateProcessList := func() {
		defer wg.Done()
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()

		for {
			<-tick.C
			procs, err := process.Processes()
			if err != nil {
				fmt.Printf("Error getting processes: %v\n", err)
				return
			}

			processData := make([]struct {
				proc *process.Process
				cpu  float64
				mem  uint64
				name string
				user string
			}, 0)

			for _, proc := range procs {
				cpu, _ := proc.CPUPercent()
				memInfo, _ := proc.MemoryInfo()
				name, _ := proc.Name()
				user, err := proc.Username()
				if err != nil {
					user = "N/A"
				}

				processData = append(processData, struct {
					proc *process.Process
					cpu  float64
					mem  uint64
					name string
					user string
				}{
					proc: proc,
					cpu:  cpu,
					mem:  memInfo.RSS / (1024 * 1024), // В мегабайтах
					name: name,
					user: user,
				})
			}

			// Сортировка процессов
			sort.Slice(processData, func(i, j int) bool {
				if sortBy == "cpu" {
					return processData[i].cpu > processData[j].cpu
				} else if sortBy == "name" {
					return strings.ToLower(processData[i].name) < strings.ToLower(processData[j].name)
				} else {
					return processData[i].mem > processData[j].mem
				}
			})

			processTable.Clear()
			processTable.SetCell(0, 0, tview.NewTableCell("PID").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignCenter))
			processTable.SetCell(0, 1, tview.NewTableCell("User").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignCenter))
			processTable.SetCell(0, 2, tview.NewTableCell("CPU Usage %").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignCenter))
			processTable.SetCell(0, 3, tview.NewTableCell("Memory Usage Mb").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignCenter))
			processTable.SetCell(0, 4, tview.NewTableCell("Process Name").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignCenter))
			for i, data := range processData {
				if i >= 100 { // Показываем только топ 100 процессов
					break
				}
				processTable.SetCell(i+1, 0, tview.NewTableCell(fmt.Sprintf("%d", data.proc.Pid)).SetAlign(tview.AlignCenter))
				processTable.SetCell(i+1, 1, tview.NewTableCell(data.user).SetAlign(tview.AlignCenter))
				processTable.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprintf("%.2f", data.cpu)).SetAlign(tview.AlignCenter))
				processTable.SetCell(i+1, 3, tview.NewTableCell(fmt.Sprintf("%d", data.mem)).SetAlign(tview.AlignCenter))
				processTable.SetCell(i+1, 4, tview.NewTableCell(data.name).SetAlign(tview.AlignLeft))
			}

			app.Draw()
		}
	}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'm':
				sortBy = "memory"
				processTable.ScrollToBeginning()
			case 'c':
				sortBy = "cpu"
				processTable.ScrollToBeginning()
			case 'n':
				sortBy = "name"
				processTable.ScrollToBeginning()
			}
		}
		return event
	})

	wg.Add(4) // Увеличиваем счетчик горутин
	go updateCpuGauges()
	go updateMemGauge()
	go updateSwapGauge()
	go updateProcessList()

	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(cpuFlex, logicalCpuCount, 1, true).
		AddItem(memGauge, 1, 1, false).
		AddItem(swapGauge, 1, 1, false).
		AddItem(platform, 1, 1, false).
		AddItem(loadAvg, 1, 1, false).
		AddItem(uptime, 1, 1, false)
	leftFlexBordered := createBorderedFrame(leftFlex)

	rightFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(processTable, 0, 1, false)
	rightFlexBordered := createBorderedFrame(rightFlex)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftFlexBordered, 0, 1, false).
		AddItem(rightFlexBordered, 0, 1, false)
	if err := app.SetRoot(mainFlex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

	wg.Wait() // Ждем завершения всех горутин
}
