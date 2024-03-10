# F1Gopher

![](./imgs/session.png)

F1Gopher is a GUI frontend to the [F1Gopher library](https://github.com/f1gopher/f1gopherlib). 

It allows you to view timing, telemetry and location data and more from Formula 1 events, either live or replays of previous events.

## Features

* Replay all sessions (practices, qualifying, sprint and races) from all events from 2018 to now
* Watch data from sessions lives as they happen
* Watch data from pre-season test sessions live
* Listen to driver radio messages
* Pause and resume live sessions
* Skip forward through replay sessions
* Count down to the next session
* Web server that duplicates the timing view onto a web page

### Timing View

![](./imgs/timing.png)

* Driver position
* Segment times for each driver that show if they were faster than their previous best, faster than anyone or slower
* Drivers fastest lap for the session
* Gap to the driver in front or gap to the fastest lap (for qualifying)
* All three sector times and last lap time color to show if the time is a personal best, fastest overall or slower
* DRS open or closed and whether the car is currently within one second of the car in front and potentially able to use DRS
* Current tire being used and number of laps the tire has been used for
* Last speed when going through the speed trap
* Location of the car (on track, outlap, pitlane, stopped...)
* Segment state for the track (is the segment green, yellow or red flagged)
* Fastest sector and laptimes for anyone in that session
* For race sessions shows the estimated position after a pitstop (including gap ahead and behind to the nearest drivers). This is estimated from the time taken to drive through the pitlane plus a configurable expected pitstop time

### Track Map View

![](./imgs/track_map.png)

* An outline of the track and pitlane
* Locations of all drivers in realtime
* Location of the safety car when active

### Radio View

* Plays the drivers radio messages as they happen
* Or mute them

### Race Control Messages View

![](./imgs/race_control_messages.png)

* Displays all messages from race control

### Qualifying Session Improving View

![](./imgs/qualifying_improving.png)

* Shows how much each drivers current has improved compared to their best time and the current pole time

### Race Session Tracker View

![](./imgs/race_tracker.png)

* Configurable view to track and compare lap times between pairs of drivers
* Shows the tire compound and current gap between drivers
* Shows the past 5 laps times and whether a driver is gaining or loosing time compared to the other driver
* You can compare a driver to any other, the car infront, the car behind, the leader or their team mate