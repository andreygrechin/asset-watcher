# asset-watcher

Test

A command-line utility to fetch and forward Google Cloud advisory notifications to Slack.

[![license](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/andreygrechin/asset-watcher/blob/main/LICENSE)

## Features

- Collect organization level advisory notifications in Google Cloud
- Send notifications to Slack

## Installation

### go install

```shell
go install github.com/andreygrechin/asset-watcher@latest
```

## Usage

```shell
export ADV_NOTIF_SLACK_CHANNEL_ID=0123456789A
export ADV_NOTIF_SLACK_TOKEN=xoxb-<YOUR_SLACK_TOKEN>
export ADV_NOTIF_ORG_ID=123456789012
export ADV_NOTIF_MAX_NOTIFICATION_AGE_SECONDS=86000
export ADV_NOTIF_DEBUG=true

make all && ./bin/asset-watcher
```

## Flow Diagram

```mermaid
graph TD
    A[Start] --> B["Load Config from Env Vars"];
    B --> C["Setup Logging"];
    C --> D["Log Version Info"];
    D --> E{"Initialize Fetcher"};
    E --> F["Fetch Notifications from Google API"];
    F -- Raw Notifications --> G{"Initialize Processor"};
    G --> H["Process Notifications (Filter, Convert, Combine)"];
    H -- Processed Notifications List --> K{"Initialize Slack Notifier"};
    K --> L{"Loop through Processed Notifications"};
    L -- Each Notification --> M["Format Slack Message"];
    M --> N["Send to Slack Channel"];
    N -- Log Success/Failure --> L;
    L -- All Sent --> O{"Any Send Errors?"};
    O -- Yes --> P["Exit(1)"];
    O -- No --> Q["Exit(0)"];

    subgraph Error Handling
        B -- Error --> P;
        E -- Error --> P;
        F -- Error --> P;
        H -- Error --> P;
        N -- Send Error --> O;
    end

    subgraph Components
        B --- config.go;
        C --- logger.go;
        E --- fetcher.go;
        F --- fetcher.go;
        G --- processor.go;
        H --- processor.go;
        H --- utils.go;
        K --- slack.go;
        M --- slack.go;
        N --- slack.go;
    end
```

## License

This project is licensed under MIT licenses —  [MIT License](LICENSE).

`SPDX-License-Identifier: MIT`
