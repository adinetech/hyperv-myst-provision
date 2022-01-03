/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	hyperv_wmi2 "github.com/mysteriumnetwork/hyperv-node/hyperv-wmi"
	transport2 "github.com/mysteriumnetwork/hyperv-node/service/daemon/transport"
	"github.com/rs/zerolog/log"
)

// Daemon - vm helper process.
type Daemon struct {
	mgr              *hyperv_wmi2.Manager
	importInProgress bool
}

// New creates a new daemon.
func New(manager *hyperv_wmi2.Manager) Daemon {
	return Daemon{mgr: manager}
}

// Start supervisor daemon. Blocks.
func (d *Daemon) Start(options transport2.Options) error {
	return transport2.Start(d.dialog, options)
}

// dialog talks to the client via established connection.
func (d *Daemon) dialog(conn io.ReadWriter) {
	scan := bufio.NewScanner(conn)

	answer := responder{conn}
	for scan.Scan() {
		line := scan.Bytes()
		log.Debug().Msgf("> %s", line)

		m := make(map[string]interface{}, 0)
		err := json.Unmarshal([]byte(line), &m)
		fmt.Println(m, err)
		op := strings.ToLower(m["cmd"].(string))

		switch op {
		case commandVersion:
			answer.ok_(nil)

		case commandPing:
			answer.pong_()

		case commandStopVM:
			err := d.mgr.StopVM()
			if err != nil {
				log.Err(err).Msgf("%s failed", commandStopVM)
				answer.err_(err)
			} else {
				answer.ok_(nil)
			}

		case commandStartVM:
			err := d.mgr.StartVM()
			if err != nil {
				log.Err(err).Msgf("%s failed", commandStartVM)
				answer.err_(err)
			} else {
				answer.ok_(nil)
			}

		case commandImportVM:
			reportProgress, _ := m["report-progress"].(bool)

			if d.importInProgress {
				// prevent parallel runs of import-vm
				answer.err_(errors.New("in progress"))
			} else {
				d.importInProgress = true
				err = d.mgr.ImportVM(hyperv_wmi2.ImportOptions{
					Force:                true,
					VMBootPollSeconds:    5,
					VMBootTimeoutMinutes: 5,
					KeystoreDir:          "",
				}, func(progress int) {
					if reportProgress {
						answer.progress_(commandImportVM, progress)
						//fmt.Println("Progress >>>", progress)
					}
				})
				if err != nil {
					log.Err(err).Msgf("%s failed", commandImportVM)
					answer.err_(err)
				} else {
					answer.ok_(nil)
				}
				d.importInProgress = false
			}

		case commandGetKvp:
			//err = d.mgr.GetGuestKVP()
			//if err != nil {
			//	log.Err(err).Msgf("%s failed", commandImportVM)
			//	answer.err_(err)
			//} else {
			answer.ok_(d.mgr.Kvp)
			//}

		}
	}
}
