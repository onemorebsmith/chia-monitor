# chia-monitor
WIP monitor that monitors the logs generate by the chia client and exposes the data to grafana using Prometheus. Also incorps scheduling and creating of more chia plots as well as moving plots to staging/farm directories.  

# grafana output

![Alt text](https://i.imgur.com/HkBFB6W.png "Grafana")


# Todo:
- Containerize the monitor
- Call chia rpc directly instead of scraping the log files/directories
- Windows support (syscalls to replace df/disk/memstat)
- MadMax plotter support
