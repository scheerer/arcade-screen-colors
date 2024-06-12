Sends screen color to RGB light bulbs.

Limitations:

* _Currently only supports running on Windows._
* _Currently only supports LIFX RGB light bulbs._

Polls to discover LIFX bulbs on the network. Once there are bulbs found, it will 
begin capturing the screen, computing the colors, and send to the lights.

_Occasionally the bulbs may disconnect, the application will eventually recover. If not, try removing power from the bulb(s) and turning them back on._

## Usage

Copy the .exe file as well as the sample StartScreenColors.bat file to a directory on your computer.

Modify the .bat file to point to the path you just copied the files to.

Edit the .bat file as needed. The defaults should work in most cases.

#### LaunchBox

If running this on a system with LaunchBox/BigBox you can add the .bat file as a startup program in LaunchBox.

Enjoy!

### Development Running

`go run cmd/main.go`

### Building

**Currently only supports Windows execution**

From Windows:
`go build -o bin/arcade-screen-colors.exe cmd/main.go`

From Other:
`GOOS=windows GOARCH=amd64 go build -o bin/arcade-screen-colors.exe cmd/main.go`
