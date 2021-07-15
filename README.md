# chia-monitor
WIP monitor that monitors the logs generate by the chia client and exposes the data to grafana using Prometheus. Also incorps scheduling and creating of more chia plots as well as moving plots to staging/farm directories. Currently tested in Ubuntu only but should work in other linux distros. Will not work in windows currently due to relying on Linux-specific calls and file structures. 

# install
- Requires go & docker installed
- clone repo
- configure the monitor
  - `cp config_example.yaml config.yaml`
  - Set ChiaPath to the location of your local `chia-blockchain` repo (ie where the activate script is)
  - Configure DriveMonitor (optional)
  - Configure Plotter (optional/experimental)
  - Configure UHaul (optional)
- launch services:
  - `./launch_grafana.sh;./launch_prom.sh`
  - `./launch_monitor.sh`
This should launch grafana/prom/monitor and expose a local grafan instance on localhost:3000. Grafana and prom are both configured using persistent storage volumes, so data will be retained between launches. At this point you should register prom with grafana (grafana->configuration->data sources). Prometheous should be active on port 9090 and automatically start scraping the local monitor instance every 15s. 


# grafana output
You can get an output similar to this if you import the grafana json export in `grafana/chia_dash.json` or configure your own using the metrics exposed to prom. 
![Alt text](https://i.imgur.com/HkBFB6W.png "Grafana")

# Features
## Drive monitoring
This feature monitors your staging and final plot paths and provides metrics such as disk activity (for temp paths) and plot counts for staging/final directories. 
## Uhaul
Uhaul monitors any drives listed as `StagingPaths` drives and moves finished plots directories listed in `FinalPaths`. Uhaul maintains an internal state so it will never attempt to have more than one file being transferred to a single drive at a time, but will allow transfers to multiple drives at once. This keeps the transfer speeds high and keeps from bogging the drive I/O rates down. Internally, UHaul uses native rysnc for reliablilty. Once transferred successfully, uhaul removes the file from staging.
## Plotter
The plotter part of chia-monitor allows for the creation of new plots in an organized manner. Currently this uses the default chia plotter from the chia-blockchain repo, but monitors the output of the plotting system to properly space and sequence plots as desired from the user. Check the `config_example.yaml` for all the options allowed here. This also supports the new portable plot format.

# Todo:
- Containerize the monitor
- Automagically import granfana config & chia_dash export file
- Call chia rpc directly instead of scraping the log files/directories
- Windows support (syscalls to replace df/disk/memstat)
- MadMax plotter support
