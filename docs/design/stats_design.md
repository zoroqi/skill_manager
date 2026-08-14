# 統計功能設計文檔 (Statistics Feature Design)

## 1. 概述 (Overview)

為了讓用戶清楚了解各個 Skill (技能) 與 Group (技能組) 在本地各項目及全局環境中的實際引用與使用頻率，本功能將讀取 `summary.json` 中登記的所有 context（包含全局 `global.json` 與各項目中的 `.skills_manager.json`），進行聚合分析並輸出統計報表。

## 2. 統計口徑與計算邏輯 (Metrics & Calculation Logic)

### 2.1 數據來源
1. **全局配置目錄中的 `summary.json`**：
   - 讀取 `summary.Contexts` 列表，獲取所有已記錄的 context 配置路徑（`config_path`）。
2. **各 Context 配置檔案 (`.skills_manager.json` / `global.json`)**：
   - 若 `config_path` 文件存在且格式合法，讀取該 Context 的 `ContextConfig`：
     - `ActiveSkills`: 用戶手動啟用的技能列表。
     - `ActiveGroups`: 啟用的技能組列表。
     - `InactiveSkills`: 屏蔽的技能列表。
   - 若檔案不存在或損壞，安全跳過並記錄提示/忽略，不中斷整體統計。
3. **全局技能庫 (`skills.json`) 與技能組庫 (`groups.json`)**：
   - 用於獲取所有已知技能與群組的全量列表及其名稱映射。

### 2.2 統計口徑定義

#### A. Group 使用次數統計 (Group Usage Count)
- **統計對象**：實際被已記錄 context 引用的技能群組（僅納入使用次數 `count > 0` 的項目，未使用的群組不參與展示）。
- **計數規則**：遍歷所有有效的 context 配置，只要該 context 的 `ActiveGroups` 中包含該 group（無論是否指定特定 agent），該 group 的計數 $+1$（即「被多少個環境/項目所使用」）。
- **排序規則**：按 `count` 降序排列；若次數相同則按 `group name` 字典序升序排列。

#### B. Skill 使用次數統計 (Skill Usage Count)
- **統計對象**：實際在 context 中生效的技能（僅納入使用次數 `count > 0` 的項目，未使用的技能不參與展示）。
- **計數規則**：
  - 在每個有效 context 中，計算該 context 下的**生效技能 (Effective Skills)**：
    - 包含用戶直接添加的 `ActiveSkills`；
    - 包含從已啟用的 `ActiveGroups` 中展開的技能（排除掉在 `InactiveSkills` 中被屏蔽的技能）；
  - 若某技能在該 context 中生效（至少在一個 Agent 中生效），則該技能的計數 $+1$。
- **排序規則**：按 `count` 降序排列；若次數相同則按 `skill name` 字典序升序排列。

## 3. CLI 命令設計 (Command Structure)

提供 `stat`（別名 `stats`, `statistics`）主命令及其子命令：

### 3.1 命令列表
1. **`skill_manager stat`**
   - 默認展示所有統計報表（依序展示 Skill 使用次數表與 Group 使用次數表）。
2. **`skill_manager stat skill`** (別名: `skills`)
   - 僅展示已使用的單個 Skill 使用次數表。
3. **`skill_manager stat group`** (別名: `groups`)
   - 僅展示已使用的 Group 使用次數表。

### 3.2 輸出報表格式

採用對 CLI 友好的簡潔列對齊格式輸出：

#### Skill 統計報表
```text
Skill Name    Count
golang        10
docker        5
```

#### Group 統計報表
```text
Group Name    Count
backend       8
devops        3
```

## 4. 架構設計與代碼分層 (Architecture & Layering)

嚴格遵守現有項目的分層規範：
```
cmd/  ──→  app  ──→  store
 ↑          ↑
 └──────────┴────── types
```

### 4.1 `internal/types.go`
新增統計相關的數據結構：
```go
type StatItem struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}

type StatsReport struct {
    Skills []StatItem `json:"skills"`
    Groups []StatItem `json:"groups"`
}
```

### 4.2 `internal/app/stats.go`
在 `App` 結構體上實現業務統計方法：
- `GetSkillStats() ([]internal.StatItem, error)`
- `GetGroupStats() ([]internal.StatItem, error)`
- `GetStatsReport() (*internal.StatsReport, error)`

**核心流程**：
1. 通過 `store.LoadSummaryConfig()` 獲取所有 context。
2. 遍歷 context，調用 `a.GetStatus(dir, global)` 統一載入各 context 配置及其對應的技能和群組庫。
3. 計算各 context 中生效的技能集合與啟用的技能組集合。
4. 累加計數、過濾掉未使用的項目（僅保留 `count > 0`）、組裝 `StatItem` 列表並進行排序。

### 4.3 `internal/cmd/stat.go`
- 註冊 `statCmd`、`statSkillCmd`、`statGroupCmd`。
- 負責調用 `application.GetStatsReport()` / `application.GetSkillStats()` / `application.GetGroupStats()`。
- 使用 `text/tabwriter` 以 CLI 友好的列對齊格式打印結果到標準輸出。

## 5. 測試策略 (TDD)

1. **單元測試 (`internal/app/stats_test.go`)**：
   - 測試無 context 時的空統計與 0 計數展示。
   - 測試多個 context（包含 `global.json` 和項目 `.skills_manager.json`）下的計數準確性。
   - 測試直接添加技能 (ActiveSkills) 與群組展開技能 (ActiveGroups) 的計數邏輯。
   - 測試 `InactiveSkills` 屏蔽群組技能時的扣除邏輯。
   - 測試不存在的 context 文件容錯（不崩潰、正常跳過）。
   - 測試排序邏輯（count 降序，name 升序）。

2. **CLI 整合測試 (`internal/cmd/stat_test.go`)**：
   - 測試 `stat`, `stat skill`, `stat group` 的命令行輸出格式與內容。

## 6. 開發計劃 (Development Plan)

- [x] 步驟 1: 在 `internal/types.go` 中定義統計數據結構。
- [x] 步驟 2: 編寫 `internal/app/stats_test.go` 測試用例。
- [x] 步驟 3: 實現 `internal/app/stats.go` 業務邏輯，確保 app 單元測試通過。
- [x] 步驟 4: 編寫 `internal/cmd/stat.go` 及 `internal/cmd/stat_test.go`。
- [x] 步驟 5: 運行全量測試 `go test ./...` 確保無回歸。
- [x] 步驟 6: 回顧與總結。

## 7. 回顧與總結 (Review & Summary)

本次開發順利完成了「統計功能」的設計與實現：
1. **分層明確**：數據模型定義在 [`internal/types.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/types.go)，業務統計計算收攏在 [`internal/app/stats.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/stats.go)，CLI 命令行交互與 Markdown 表格渲染封裝在 [`internal/cmd/stat.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/cmd/stat.go)。
2. **統計口徑嚴謹**：
   - 遍歷 `summary.json` 中的所有 context（包含 `global.json` 及各項目的 `.skills_manager.json`）。
   - 對 Skill 使用統計綜合考慮了用戶手動啟用的技能與 Group 展開技能，並正確扣除了 `InactiveSkills` 屏蔽項。
   - 容錯處理：遇到不存在或格式損壞的 context 配置時能夠安全跳過。
   - 結果排序：按使用次數降序排列，次數相同時按名稱字典序升序排列。
3. **命令豐富友好**：
   - 支援 `skill_manager stat` 同時展示 Skill 與 Group 報表。
   - 支援 `skill_manager stat skill` 與 `skill_manager stat group` 子命令單獨查詢。
   - 支援別名 `stats`, `statistics`, `skills`, `groups`。
4. **高覆蓋率測試**：
   - 包含單元測試 [`internal/app/stats_test.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/stats_test.go) 與 CLI 整合測試 [`internal/cmd/stat_test.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/cmd/stat_test.go)，全量測試通過。

