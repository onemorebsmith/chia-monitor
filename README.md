# chia-monitor
WIP monitor that monitors the logs generate by the chia client and exposes the data to grafana using Prometheus. Also incorps scheduling and creating of more chia plots as well as moving plots to staging/farm directories.  

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


# Todo:
- Containerize the monitor
- Automagically import granfana config & chia_dash export file
- Call chia rpc directly instead of scraping the log files/directories
- Windows support (syscalls to replace df/disk/memstat)
- MadMax plotter support
