/*
   Raven Network Discovery and Monitoring
   Copyright (C) 2019 John{at}Orthoefer{dot}org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.

*/
package license

import (
	"fmt"
	"io"
	"log"
	"os"
)

const version = "$Id$"

func buildLicense() (rtn []string) {
	return append(rtn,
		"Raven Network Discovery and Monitoring",
		version,
		fmt.Sprintf("Copyright (C) %d  %s\n", 2019,
			"John{at}Orthoefer{dot}org"),
		"This program comes with ABSOLUTELY NO WARRANTY.",
		"This is free software, and you are welcome to redistribute",
		"it under certain conditions.  For details see COPYING text file.")
}

func LogLicense() {
	for _, v := range buildLicense() {
		log.Printf("%s", v)
	}
}

func licenseOutput(w io.Writer) {
	for _, v := range buildLicense() {
		fmt.Fprintf(w, "%s", v)
	}
}

func PrintLicense() {
	licenseOutput(os.Stdout)
}

func ErrLicense() {
	licenseOutput(os.Stderr)
}
