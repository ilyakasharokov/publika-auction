package csv_service

import (
	"encoding/csv"
	"os"
	"publika-auction/internal/app/clients-repo"
	"strings"
)

type CsvService struct {
}

func New() *CsvService {
	return &CsvService{}
}

func (cs *CsvService) Read() (clients map[string]clients_repo.Client, err error) {
	clients = make(map[string]clients_repo.Client)
	file := "clients_data-23.02.2024r.csv"
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return clients, err
	}
	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return clients, err
	}
	for _, line := range lines {
		splited := strings.Split(line[0], ";")
		name := splited[0]
		phone := strings.Replace(splited[1], "'", "", 1)
		email := splited[2]
		if phone != "" {
			if cl, ok := clients[phone]; ok {
				cl.Names = append(cl.Names, name)
				continue
			}
			clients[phone] = clients_repo.Client{Name: name, Phone: phone, Email: email, Names: make([]string, 0)}
		}
	}
	return clients, nil
}
