# TUI Design Document

## 概述 (Overview)
`skill_manager` 包含一個基於文本的使用者介面 (TUI)，使用 `charmbracelet/bubbletea` 框架開發。TUI 旨在提供一個免記憶 CLI 參數、互動式的操作環境來管理本地的專案技能、全域技能、遠端技能註冊 (Registers) 以及技能群組 (Groups)。

## 架構設計 (Architecture Design)
TUI 系統主要實作於 `internal/tui` 目錄中，嚴格遵循 Elm 架構 (Model, Update, View)，並廣泛採用 `charmbracelet/bubbles` 官方組件庫以降低複雜度。

### 1. 模型 (Model)
- **檔案:** `internal/tui/model.go`
- **結構:** `Model` 結構體保存了整個 TUI 的狀態。
  - `state` (`ViewState`): 控制當前畫面的模式，包含儀表板 (`DashboardView`)、輸入表單 (`FormView`) 與詳細資訊視圖 (`DetailView`)。
  - `active` (`Panel`): 指示當前獲得焦點的面板（0: 專案, 1: Register, 2: 全域, 3: Group）。
  - **Bubbles 組件**:
    - `projectList`, `globalList`, `registerList`, `groupList` (`list.Model`): 分別負責四個區域的列表展示、搜尋過濾與導覽。
    - `formInputs` (`[]textinput.Model`): 管理表單中的多個輸入框，支援焦點切換。
    - `detailViewport` (`viewport.Model`): 用於展示長篇幅的技能詳細資訊 JSON，支援捲動。

### 2. 資料模型與介面實作 (Data Model & Interface Implementation)
- **檔案:** `internal/tui/list_items.go`
- 為了與 `bubbles/list` 相容，專案中的資料結構 (`ResolvedSkill`, `RegisterItem`, `GroupItem`) 均實作了 `list.Item` 介面：
  - `Title()`, `Description()`, `FilterValue()`。
- **展示委派 (`skillDelegate`)**:
  - 位於 `internal/tui/delegate.go`，負責自定義列表項目的渲染。
  - **緊湊布局**: 採用單行渲染，優化大螢幕空間利用率。
  - **層次化展示 (Hierarchy)**: 在 Register 與 Group 窗口中，父級項目加粗顯示，子技能則透過縮排 (`%-20s`) 展示於 `skill_name` 欄位下，建立清晰的隸屬關係。

### 3. 更新與事件處理 (Update)
- **檔案:** `internal/tui/update.go`
- **邏輯委派**: `Update` 函式將大部分按鍵事件委派給當前獲得焦點的組件（如當前的 `list` 或 `textinput`）。
- **按鍵綁定 (Keybindings):**
  - **導覽**:
    - `Tab` / `Shift+Tab`: 在四個面板間循環切換焦點。
    - `Up` / `Down` (或 `j` / `k`): 在列表內移動。
  - **操作**:
    - `a`: 進入 `FormView` 新增 Registry、Group 或 Skill。
    - `d`: 刪除目前選中的項目。
    - `Enter`: 
      - 在 Dashboard 進入選中技能的詳情頁。
      - 在 `FormView` 跳轉至下一欄位或送出表單。
    - `/`: 開啟當前列表的內建搜尋過濾功能。
  - **詳情頁快捷鍵**: 
    - `p` / `g`: 直接將當前查看的技能加入專案或全域配置。

### 4. 視圖渲染 (View)
- **檔案:** `internal/tui/view.go`
- **佈局與樣式**: 採用 `charmbracelet/lipgloss` 進行樣式定義與佈局拼接。
  - 主畫面 (`dashboardView`) 採用 2x2 網格。
  - **高亮機制**: 獲得焦點的面板會以亮綠色 (`Color 2`) 粗邊框顯示，未獲焦點則顯示灰色細邊框。
  - **自定義標題**: 每個面板頂部均設有固定的列標題（如 `Name`, `Register`, `Agent`, `Source`），確保資料對齊。

## 擴充性 (Extensibility)
- **新組件**: 透過引入官方 `bubbles` 組件，新增功能（如分頁、搜尋）變得更加簡單且穩定。
- **表單擴展**: 只需在 `setupForm` 中增加 `textinput` 實例，並在 `submitForm` 讀取其 `Value()` 即可。
