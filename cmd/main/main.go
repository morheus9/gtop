// Demo code for the bar chart primitive.
package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
	"github.com/shirou/gopsutil/process"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func createBorderedFrame(inner tview.Primitive) *tview.Frame {
	frame := tview.NewFrame(inner).
		SetBorders(1, 1, 1, 1, 1, 1) // Устанавливаем одинарную рамку со всех сторон
	frame.SetBorder(true).SetBorderColor(tcell.ColorLightSkyBlue)
	return frame
}
func main() {
	app := tview.NewApplication()
	// Получаем количество процессоров
	logicalCpuCount, _ := cpu.Counts(true)
	// Получаем количество процессоров

	// Создаем массив для индикаторов CPU
	cpuGauges := make([]*tvxwidgets.UtilModeGauge, logicalCpuCount)

	// Создаем индикаторы для каждого процессора
	for i := 0; i < logicalCpuCount; i++ {
		cpuGauge := tvxwidgets.NewUtilModeGauge()
		cpuGauge.SetLabel(fmt.Sprintf(" CPU %d ", i))
		cpuGauge.SetLabelColor(tcell.ColorLightSkyBlue)
		cpuGauge.SetBorder(false)
		cpuGauges[i] = cpuGauge
	}

	// Функция для обновления индикаторов CPU
	updateCpuGauges := func() {
		tick := time.NewTicker(500 * time.Millisecond)
		for {
			select {
			case <-tick.C:
				v, err := cpu.Percent(1*time.Second, true) // true для получения данных по каждому процессору
				if err != nil {
					fmt.Printf("Error getting CPU percent: %v\n", err)
					return
				}
				for i := 0; i < logicalCpuCount; i++ {
					cpuGauges[i].SetValue(v[i]) // Обновляем значение для каждого индикатора
				}
				app.Draw()
			}
		}
	}

	// memory usage gauge
	memGauge := tvxwidgets.NewUtilModeGauge()
	memGauge.SetLabel(" mem   ")
	memGauge.SetLabelColor(tcell.ColorLightSkyBlue)
	memGauge.SetRect(10, 3, 50, 3)
	memGauge.SetWarnPercentage(65)
	memGauge.SetCritPercentage(80)
	memGauge.SetBorder(false)
	updateMemGauge := func() {
		tick := time.NewTicker(500 * time.Millisecond)
		for {
			select {
			case <-tick.C:
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
	}

	// swap usage gauge
	swapGauge := tvxwidgets.NewUtilModeGauge()
	swapGauge.SetLabel(" swap  ")
	swapGauge.SetLabelColor(tcell.ColorLightSkyBlue)
	swapGauge.SetRect(10, 3, 50, 3)
	swapGauge.SetWarnPercentage(65)
	swapGauge.SetCritPercentage(80)
	swapGauge.SetBorder(false)
	updateSwapGauge := func() {
		tick := time.NewTicker(500 * time.Millisecond)
		for {
			select {
			case <-tick.C:
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
	}

	go updateCpuGauges()
	go updateMemGauge()
	go updateSwapGauge()

	cpuFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	for _, gauge := range cpuGauges {
		cpuFlex.AddItem(gauge, 1, 1, false) // Добавляем каждый индикатор в Flex-контейнер
	}

	// Создаем Flex-контейнер для левой половины
	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(cpuFlex, logicalCpuCount, 1, true). // Занимает всю высоту
		AddItem(memGauge, 1, 1, false).             // Фиксированная высота
		AddItem(swapGauge, 1, 1, false)             // Фиксированная высота

	// Создаем рамку вокруг Flex-контейнера
	leftFlexBordered := createBorderedFrame(leftFlex)

	processTable := tview.NewTable().SetBorders(true)
	// Добавляем заголовки таблицы
	processTable.SetFixed(1, 0) // Устанавливаем фиксированную высоту для заголовков таблицы

	processTable.SetCell(0, 0, tview.NewTableCell("Process Name").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignLeft))
	processTable.SetCell(0, 1, tview.NewTableCell("PID").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignLeft))
	processTable.SetCell(0, 2, tview.NewTableCell("CPU Usage (%)").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignLeft))
	processTable.SetCell(0, 3, tview.NewTableCell("Memory Usage (%)").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignLeft))
	processTable.SetCell(0, 4, tview.NewTableCell("Time").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignLeft))

	sortBy := "cpu" // Переменная для отслеживания текущего порядка сортировки

	updateProcessList := func() {
		tick := time.NewTicker(1 * time.Second)
		defer tick.Stop() // Останавливаем тикер при выходе из функции
		for {
			<-tick.C
			procs, err := process.Processes()
			if err != nil {
				fmt.Printf("Error getting processes: %v\n", err)
				return
			}

			// Сортируем процессы в зависимости от текущего порядка сортировки
			sort.Slice(procs, func(i, j int) bool {
				if sortBy == "cpu" {
					cpuI, _ := procs[i].CPUPercent()
					cpuJ, _ := procs[j].CPUPercent()
					return cpuI > cpuJ
				} else {
					memI, _ := procs[i].MemoryPercent()
					memJ, _ := procs[j].MemoryPercent()
					return memI > memJ
				}
			})

			// Очищаем таблицу
			processTable.Clear()

			// Добавляем заголовки
			processTable.SetCell(0, 0, tview.NewTableCell("Process Name").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignCenter))
			processTable.SetCell(0, 1, tview.NewTableCell("PID").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignCenter))
			processTable.SetCell(0, 2, tview.NewTableCell("CPU Usage %").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignCenter))
			processTable.SetCell(0, 3, tview.NewTableCell("Memory Usage %").SetTextColor(tcell.ColorLightSkyBlue).SetAlign(tview.AlignCenter))

			// Ограничиваем количество отображаемых процессов
			for i, proc := range procs {
				if i >= 30 { // Показываем только топ 20 процессов
					break
				}
				name, _ := proc.Name()
				pid := proc.Pid // Получаем PID процесса (просто используем переменную)
				cpuPercent, _ := proc.CPUPercent()
				memPercent, _ := proc.MemoryPercent()
				// Получаем время работы процесса (например, в формате "hh:mm:ss")
				// Здесь предполагается, что у вас есть функция для получения времени работы
				// В противном случае, вы можете использовать proc.CreateTime() или аналогичную функцию

				processTable.SetCell(i+1, 0, tview.NewTableCell(name).SetAlign(tview.AlignLeft))
				processTable.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%d", pid)).SetAlign(tview.AlignCenter))
				processTable.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprintf("%.2f", cpuPercent)).SetAlign(tview.AlignCenter))
				processTable.SetCell(i+1, 3, tview.NewTableCell(fmt.Sprintf("%.2f", memPercent)).SetAlign(tview.AlignCenter))
			}

			app.Draw() // Обновляем отображение приложения
		}
	}

	// Устанавливаем обработчик ввода
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'm': // Нажата клавиша "m"
				sortBy = "memory"
			case 'c': // Нажата клавиша "c"
				sortBy = "cpu"
			}
		}
		return event
	})

	// Запускаем обновление списка процессов в горутине
	go updateProcessList()
	// Создаем Flex-контейнер для правой половины (пока пустой)

	rightFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(processTable, 0, 1, true) // true означает, что таблица будет расширяться
		// Добавляем Box в Flex-контейнер

	rightFlexBordered := createBorderedFrame(rightFlex)

	// Создаем основной Flex-контейнер
	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftFlexBordered, 0, 1, true). // Занимает всю высоту, но только левую половину ширины
		AddItem(rightFlexBordered, 0, 1, true) // Занимает всю высоту правой половины
	// Устанавливаем основной Flex-контейнер как корень приложения
	if err := app.SetRoot(mainFlex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

}
