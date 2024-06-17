# Valid values are: [AVERAGE, SQUARED_AVERAGE, MEDIAN, MODE]
set COLOR_ALGO=AVERAGE

# How often to capture the screen (should generally be greater than 50ms due to screen capture latency)
set CAPTURE_INTERVAL=80ms

# Valid values are: [LIFX]
set LIGHT_TYPE=LIFX

# Name of LIFX group to control
set LIGHT_GROUP_NAME=ARCADE

# Adjust maximum brightness of the lights between 0 and 1. 1 is full brightness. (makes screen flashes or white screens quite bright)
set MAX_BRIGHTNESS=0.65

# Adjust minimum brightness of the lights between 0 and 1. 0 is the light turned off.
set MIN_BRIGHTNESS=0

# Adjust PIXEL_GRID_SIZE to increase performance or accuracy. Lower values are slower but more accurate. 1 being the most accurate.
set PIXEL_GRID_SIZE=5

# Adjust SCREEN_NUMBER to target a different screen. 0 is the primary screen.
set SCREEN_NUMBER=0

G:\Tools\arcade-screen-colors.exe