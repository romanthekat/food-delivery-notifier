package delivio

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"
)

type logData struct {
	restaurantId, status, restCoor, destCoor, courierCoor string
}

func writeLogs(data *logData) {
	f, err := os.OpenFile("data_raw.csv", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	w := csv.NewWriter(f)
	err = w.Write([]string{
		time.Now().Format("2006-01-02 15:04:05"),
		data.restaurantId,
		data.status,
		data.restCoor,
		data.destCoor,
		data.courierCoor,
	})

	if err != nil {
		fmt.Printf("failed to write data: %s\n", err)
	}

	w.Flush()
}
