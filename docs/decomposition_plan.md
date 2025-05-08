# Decomposition Plan for Advisory Notifications Processor

This document outlines a proposed decomposition for the Go application that fetches Google Cloud Advisory Notifications and potentially sends them to Slack, applying SOLID principles.

## Analysis & SOLID Violations

The current implementation in `main.go` exhibits the following SOLID principle violations:

1. **Single Responsibility Principle (SRP):**
    * The `main` function violates SRP by handling multiple distinct responsibilities:
        * Data transformation (HTML to Markdown conversion).
        * Outputting results directly to the console.
        * Unconditionally calling `sendSlackMessage`.
    * The `sendSlackMessage` function mostly adheres to SRP (sending a Slack message), but its current integration (called unconditionally) and hardcoded message content are problematic. It also handles its own configuration loading (Slack token/channel).

2. **Open/Closed Principle (OCP):**
    * The current structure is not easily open for extension without modification. For example, adding a new notification destination (like email) or supporting different input sources would require changing the existing `main` function significantly.

3. **Dependency Inversion Principle (DIP):**
    * The `main` function directly depends on concrete implementations:
        * `advisorynotifications.NewClient`
        * `slack.New`
        * `htmltomarkdown.ConvertString`
    * High-level modules should depend on abstractions, not concrete low-level implementations.

## Proposed Decomposition Plan

To address these violations and improve modularity, testability, and maintainability, the following decomposition is proposed:

1. **Data Processing/Transformation:**
    * **Responsibility:** Convert notification message bodies from HTML to Markdown format.
    * **Component:** A `HtmlToMarkdownConverter` component (could be a simple function or a struct wrapping the library).
    * **SOLID:** Adheres to SRP.

2. **Notification Formatting:**
    * **Responsibility:** Format the fetched and processed notification data into a string representation suitable for a specific output channel (e.g., console log, Slack message).
    * **Component:** A `NotificationFormatter` interface with concrete implementations (e.g., `ConsoleFormatter`, `SlackMessageFormatter`). Each formatter takes notification data and returns a formatted string.
    * **SOLID:** Adheres to SRP and OCP (easy to add new formats).

3. **Notification Dispatching:**
    * **Responsibility:** Send the formatted notification data to its intended destination(s).
    * **Component:** A `Notifier` interface with concrete implementations:
        * `ConsoleNotifier`: Prints the formatted string to the console.
        * `SlackNotifier`: Takes the formatted string (ideally from `SlackMessageFormatter`), Slack configuration details (from `Config`), and uses the Slack client to send the message. This replaces the logic of the original `sendSlackMessage` but operates on actual notification data.
    * **SOLID:** Adheres to SRP, OCP (easy to add new destinations like email), and DIP (orchestration logic depends on the `Notifier` interface).

4. **Orchestration (`main` function):**
    * **Responsibility:** Initialize all components and coordinate the overall application flow: Load Config -> Fetch Notifications -> (Process/Transform) -> Format -> Dispatch.
    * **SOLID:** The `main` function becomes much simpler, primarily responsible for application startup and orchestrating the interaction between the other components, thus adhering better to SRP.

## Visual Representation (Mermaid Diagram)

```mermaid
graph TD
    subgraph Configuration
        A[LoadConfig Function] --> B(Config Struct);
    end

    subgraph Fetching
        C[AdvisoryNotification API Client] --> D(NotificationFetcher Service);
        B -- Org ID --> D;
    end

    subgraph Processing
        E[HTML to Markdown Lib] --> F(HtmlToMarkdownConverter);
        G[Raw Notification Data] --> F;
    end

    subgraph Formatting
        H{NotificationFormatter Interface};
        I[ConsoleFormatter] --> H;
        J[SlackMessageFormatter] --> H;
        F -- Processed Data --> H;
    end

    subgraph Dispatching
        K{Notifier Interface};
        L[ConsoleNotifier] --> K;
        M[SlackNotifier] --> K;
        H -- Formatted Data --> K;
        B -- Slack Config --> M;
    end

    subgraph Orchestration
        N(main Function) --> A;
        N --> D;
        N --> F;
        N --> H;
        N --> K;
        D -- Raw Data --> G;
    end

    style Configuration fill:#f9f,stroke:#333,stroke-width:2px
    style Fetching fill:#ccf,stroke:#333,stroke-width:2px
    style Processing fill:#ffc,stroke:#333,stroke-width:2px
    style Formatting fill:#cfc,stroke:#333,stroke-width:2px
    style Dispatching fill:#cff,stroke:#333,stroke-width:2px
    style Orchestration fill:#eee,stroke:#333,stroke-width:2px
