# TUI Design Document

## 概述 (Overview)
`skill_manager` 包含一個基於文本的使用者介面 (TUI)，使用 `charmbracelet/bubbletea` 框架開發。TUI 旨在提供一個免記憶 CLI 參數、互動式的操作環境來管理本地的專案技能、全域技能、遠端技能註冊 (Registers) 以及技能群組 (Groups)。

## 架構設計 (Architecture Design)
TUI 系統主要實作於 `internal/tui` 目錄中，嚴格遵循 Elm 架構 (Model, Update, View)。

### 1. 模型 (Model)
- **檔案:** `internal/tui/model.go`
- **結構:** `Model` 結構體保存了整個 TUI 的狀態。
  - `state` (`ViewState`): 控制當前畫面的模式，包含儀表板 (`DashboardView`)、輸入表單 (`FormView`) 與詳細資訊視圖 (`DetailView`)。
  - `active` (`Panel`): 指示當前獲得焦點的面板（0: 專案, 1: Register, 2: 全域, 3: Group）。
  - `selectedRow`: 一個長度為 4 的陣列，分別記錄四個面板目前選定的行數游標 (`idx`)。
  - `searchMode` & `searchQuery`: 用於控制並暫存 Register 和 Group 面板的快速檢索狀態。
  - `projectSkills`, `globalSkills`: 儲存經過解析的當前/全域技能使用狀態 (`ResolvedSkill`)。
  - `registerItems`, `groupItems`: 採用扁平化陣列結構儲存 Register 和 Group 資訊。

### 2. 資料解析與扁平化 (Data Resolution & Flattening)
為了讓複雜的巢狀資料能在 TUI 中被輕易的捲動和選中，我們採用了「資料扁平化」策略：
- **`ResolvedSkill`**: 將 `app.GetStatus` 取得的狀態進行解析，合併來源 (User 或特定 Group) 和有效的 Agent，計算出 `ID`, `Register`, `Agents`, `Source` 供表格化展示。
- **`RegisterItem` & `GroupItem`**: 將一個 Register/Group 與其底下的多個 Skill 轉換為平坦的項目陣列。每個項目包含一個布林值旗標 (`IsRegistry` / `IsGroup`) 來區分這行是標題列還是子技能列，這樣游標就能夠逐行精確選中。

### 3. 更新與事件處理 (Update)
- **檔案:** `internal/tui/update.go`
- **按鍵綁定 (Keybindings):**
  - **導覽**:
    - `Tab` / `Shift+Tab`: 在四個面板間循環切換焦點。
    - `j` / `k` (或 上下方向鍵): 在當前獲得焦點的面板內上下移動游標。
  - **操作**:
    - `a`: 觸發新增操作，進入 `FormView`。
    - `d`: 刪除當前游標所選中的項目。
    - `Enter`: 
      - 在 `FormView` 中送出表單。
      - 在 Register/Group 面板中若選中具體的 Skill，則進入 `DetailView` 以格式化的 JSON 展示技能詳細資訊。
    - `/`: 進入檢索模式 (`searchMode`)，針對 Register 或 Group 面板進行快速名稱過濾。
  - **全域**: `Ctrl+C` 或 `q` 退出程式；`Esc` 取消表單、退出詳情頁或退出搜尋模式。

### 4. 視圖渲染 (View)
- **檔案:** `internal/tui/view.go`
- **佈局:**
  - 主畫面 (`dashboardView`) 採用 2x2 的網格佈局。
  - 左半邊顯示：Project Skills (上)、Global Skills (下)。
  - 右半邊顯示：Registries (上)、Groups (下)。
- **表格渲染 (`renderPanel`)**:
  - 每個面板會根據傳入的寬高進行計算。
  - 實作了 `truncate` 函數，當文字超過面板寬度時自動截斷以防止版面跑版。
  - 實作了**自動捲動 (Auto-scroll)** 機制：當項目數量超過面板高度 (`viewHeight`) 時，根據當前游標 (`idx`) 計算 `startIdx`，只渲染可見範圍內的項目。
  - 被選中的面板外框會以綠色高亮，當下游標所在的行則會使用 ANSI 反白 (`\033[7m`)。
- **搜尋列**: 當進入 `searchMode` 時，底部會出現實體的輸入框，即時顯示 `searchQuery`。

## 擴充性 (Extensibility)
- 若要新增新的面板，只需在 `Panel` 常數中擴充，並調整 `dashboardView` 的網格切割比例。
- 若要支援更多欄位的表單，只需在 `setupForm` 中增加 `formFields` 的定義，並在 `submitForm` 補上對應的處理邏輯即可。
