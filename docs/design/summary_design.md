# Summary 功能設計文檔 (Summary Feature Design)

## 1. 概述 (Overview)

為了讓用戶一目了然地掌握所有在 `summary.json` 中登記的項目（包括全局 `global` 環境與各本地項目）及其各自啟用的技能 (Skills) 與技能組 (Groups)，新增 `summary` 命令行功能。該功能以表格形式對齊輸出各項目的名稱、已啟用的技能列表以及已啟用的群組列表。

## 2. 數據來源與處理邏輯 (Data Sources & Processing Logic)

### 2.1 數據來源
1. **全局配置目錄中的 `summary.json`**：
   - 讀取 `summary.Contexts` 列表，獲取所有已記錄的 context 配置路徑（`config_path`）與目錄（`dir`）。
2. **各 Context 配置檔案 (`.skills_manager.json` / `global.json`)**：
   - 若 `config_path` 文件存在且格式合法，讀取該 Context 的 `ContextConfig`：
     - `Scope`: 項目作用域名稱。
     - `ActiveSkills`: 該項目中啟用的技能列表。
     - `ActiveGroups`: 該項目中啟用的技能組列表。
   - 若檔案不存在或損壞，安全跳過並記錄/忽略，不中斷整體輸出。
3. **全局技能庫 (`skills.json`) 與技能組庫 (`groups.json`)**：
   - 用於獲取技能 ID 到技能名稱 (`Skill.Name`) 的映射。

### 2.2 項目與列數據處理邏輯

#### A. 項目名稱解析 (Project Name)
- 若 `ctxSummary.Dir == "global"`，項目名稱為 `"global"`。
- 若為項目目錄，優先使用 `ctxCfg.Scope`；若 `Scope` 為空，則取目錄路徑的最後一級名稱 `filepath.Base(ctxSummary.Dir)`。

#### B. 技能列表 (Skills)
- 提取該 Context 的 `ActiveSkills`，透過 `skills.json` 映射獲取其顯示名稱（若無對應名稱則保留 `SkillUsage.ID`）。
- 進行去重並按名稱字母序 (Alphabetical Order) 升序排列。

#### C. 群組列表 (Groups)
- 提取該 Context 的 `ActiveGroups` 中的群組名稱 (`GroupUsage.Name`)。
- 進行去重並按名稱字母序 (Alphabetical Order) 升序排列。

#### D. 表格多行對齊算法 (Multi-row Alignment)
- 對於每個項目，計算所需行數：`rows = max(len(skills), len(groups), 1)`。
- 對於第 $i$ 行 ($0 \le i < rows$)：
  - **第 1 欄 (Project)**：僅在 $i = 0$ 時輸出 `project_name`，其餘行輸出空字串 `""`。
  - **第 2 欄 (Skill)**：若 $i < len(skills)$ 輸出 `skills[i]`，否則輸出空字串 `""`。
  - **第 3 欄 (Group)**：若 $i < len(groups)$ 輸出 `groups[i]`，否則輸出空字串 `""`。
- 技能與群組各自獨立展示，互不關聯，保證並排緊湊輸出。

## 3. CLI 命令設計 (Command Structure)

### 3.1 命令定義
- **主命令**：`skill_manager summary`
- **別名**：`summaries`
- **說明**：Show summary of skills and groups configured across all projects

### 3.2 輸出格式示例

使用標準 `text/tabwriter` 進行列寬動態對齊輸出：

```text
Project        Skill      Group
global         golang     backend
               docker     devops
my_project     python     backend
               react      
other_proj     docker     infra
```

## 4. 架構設計與代碼分層 (Architecture & Layering)

遵循項目既有的單向依賴分層架構：
```
cmd/  ──→  app  ──→  store
 ↑          ↑
 └──────────┴────── types
```

### 4.1 `internal/types.go`
新增數據結構：
```go
type ContextSummaryItem struct {
	ProjectName string   `json:"project_name"`
	Dir         string   `json:"dir"`
	Skills      []string `json:"skills"`
	Groups      []string `json:"groups"`
}

type SummaryReport struct {
	Items []ContextSummaryItem `json:"items"`
}
```

### 4.2 `internal/app/summary.go`
在 `App` 結構體上實現：
- `GetSummaryReport() (*internal.SummaryReport, error)`

**核心流程**：
1. 調用 `a.store.LoadSummaryConfig()` 獲取所有註冊 context。
2. 遍歷 context，調用 `a.GetStatus(dir, global)` 載入配置與技能庫。
3. 提取並排序 skills 與 groups。
4. 組裝 `ContextSummaryItem` 列表並返回 `SummaryReport`。

### 4.3 `internal/cmd/summary.go`
- 註冊 `summaryCmd`。
- 調用 `application.GetSummaryReport()`。
- 使用 `text/tabwriter` 格式化輸出各項目行。

## 5. 測試策略 (TDD)

1. **單元測試 (`internal/app/summary_test.go`)**：
   - 測試無 context 時返回空列表。
   - 測試多個項目（包含 global 與各項目）的 skills/groups 提取與去重排序。
   - 測試項目中只有 skills 或只有 groups 或兩者皆有的情況。
   - 測試不存在的 context 目錄容錯（安全跳過）。

2. **CLI 整合測試 (`internal/cmd/summary_test.go`)**：
   - 測試 `skill_manager summary` 命令輸出格式。
   - 測試多行對齊與空白欄位渲染。
   - 測試別名 `skill_manager summaries`。

## 6. 開發計劃 (Development Plan)

- [x] 步驟 1: 在 `internal/types.go` 中定義 Summary 數據結構。
- [x] 步驟 2: 編寫 `internal/app/summary_test.go` 測試用例。
- [x] 步驟 3: 實現 `internal/app/summary.go` 業務邏輯，確保 app 單元測試通過。
- [x] 步驟 4: 編寫 `internal/cmd/summary.go` 及 `internal/cmd/summary_test.go`。
- [x] 步驟 5: 運行全量測試 `go test ./...` 確保無回歸。
- [x] 步驟 6: 回顧總結並更新設計文檔。

## 7. 回顧與總結 (Review & Summary)

本次開發順利完成了「Summary 功能」的設計與實現：
1. **清晰的架構分層**：
   - 數據模型定義在 [`internal/types.go`](skill_manager/internal/types.go) (`ContextSummaryItem` 與 `SummaryReport`)。
   - 業務提取與去重排序邏輯封裝在 [`internal/app/summary.go`](skill_manager/internal/app/summary.go) (`GetSummaryReport`)。
   - CLI 命令行交互與 `tabwriter` 表格格式化輸出封裝在 [`internal/cmd/summary.go`](skill_manager/internal/cmd/summary.go)。
2. **靈活且嚴謹的展示邏輯**：
   - 遍歷 `summary.json` 中的所有 context（包含 `global` 與各項目環境）。
   - 自動映射技能名稱，並對項目內的技能與群組進行去重及字母序排序。
   - 採用多行緊湊對齊算法，技能與群組各自獨立並排輸出，首行展示項目名稱，後續行對齊展示各技能/群組。
   - 具備健壯的容錯能力：若 context 設定檔不存在或損毀，安全跳過，不中斷整體輸出。
3. **完整的測試覆蓋**：
   - 單元測試 [`internal/app/summary_test.go`](skill_manager/internal/app/summary_test.go) 涵蓋空狀態、多項目、去重排序、自定義 scope、未知技能 ID 容錯及不存在路徑處理。
   - CLI 整合測試 [`internal/cmd/summary_test.go`](skill_manager/internal/cmd/summary_test.go) 驗證了標準輸出、多行對齊與命令別名 (`summaries`)。
   - 全量測試 `go test ./...` 100% 通過。
