# Status 提示與 Sync 失效清理設計文檔 (Status Warning & Sync Cleanup Design)

## 1. 概述 (Overview)

當技能庫（`skills.json`）或技能組（`groups.json`）中的 Skill 因遠端倉庫更新、本地清理或手動移除而消失時，現有 Context 配置（專案 `.skills_manager.json` 或全局 `global.json`）的 `active_skills` 中可能仍殘留著這些已失效的技能 ID。

為了提升使用者體驗與配置自癒能力：
1. **`status` 命令**：當檢測到當前 Context 中有已失效/缺失的技能時，在狀態輸出底部給出明確警告提示，並引導使用者執行 `skill_manager sync`（或 `skill_manager sync -g`）進行清理。
2. **`sync` 命令**：在同步軟鏈的同時，主動清除 Context 配置中無效的 `active_skills`（以及不存在的 `active_groups`），並清理 Agent 目錄中的孤兒軟鏈，完成配置的自動修復與持久化。

---

## 2. 現狀問題與分析 (Current Problem & Analysis)

1. **`status` 命令缺少失效提醒**：
   - 目前 [`internal/cmd/status.go`](skill_manager/internal/cmd/status.go) 遍歷 `ctxCfg.ActiveSkills` 時，若 `skillMap[usage.ID]` 不存在則直接 `continue` 跳過。
   - 使用者無法得知哪些手動添加的技能已經失效，也無從得知為何某些技能不再生效。
2. **`sync` 命令未清理 Context 內部無效引用**：
   - 在 [`internal/app/context.go`](skill_manager/internal/app/context.go) 的 `syncContext` 中，遇到無效的 `usage.ID` 僅在內存中略過軟鏈建立，但並未從 `ctxCfg.ActiveSkills` 中剔除該記錄，也未回寫到配置文件。
   - 導致 `.skills_manager.json` 或 `global.json` 永久殘留失效的技能 ID。

---

## 3. 設計方案與業務邏輯 (Design Specification & Business Logic)

### 3.1 `status` 命令增強
1. **失效技能收集**：
   - 遍歷 `ctxCfg.ActiveSkills`，若 `usage.ID` 不存在於 `skillMap` 中，記錄至 `missingSkills` 列表。
   - 遍歷 `ctxCfg.ActiveGroups`，若 `usage.Name` 不存在於 `groupMap` 中，記錄至 `missingGroups` 列表。
2. **警告提示格式**：
   - 在標準生效技能表格下方，若存在失效技能或技能組，打印警示信息：
     ```text
     Warning: 1 active skill(s) missing or deleted from skills library:
       - 60ee599f1c6077e0eca1916b56cd4ee4 (agents: all agents)

     Please run `skill_manager sync` to clean up invalid skills.
     ```
   - 若為全局狀態（`-g` / `--global`），提示指令自動切換為 `skill_manager sync -g`。

### 3.2 `sync` 命令自愈清理
1. **Context 配置清洗**：
   - 在 `syncContext` 執行之初，過濾 `ctxCfg.ActiveSkills`，僅保留存在於 `skillMap`（即 `skills.json`）中的有效技能。
   - 過濾 `ctxCfg.ActiveGroups`，僅保留存在於 `groupMap`（即 `groups.json`）中的有效技能組。
2. **軟鏈與 Gitignore 同步**：
   - 對於已被移除的失效技能，其在 Agent 目錄下的軟鏈將自動被視為孤兒軟鏈（Orphan Symlink）並被清理。
   - 自動更新 `.gitignore`。
3. **配置持久化**：
   - `SyncContext` 在執行完 `syncContext` 後調用 `saveCtx()`，將清理後的 `ContextConfig` 寫回磁盤（`.skills_manager.json` 或 `global.json`），實現狀態自癒。

---

## 4. 架構設計與代碼分層 (Architecture & Layering)

遵循分層規範：
```
cmd/status.go, sync.go  ──→  app/context.go  ──→  store
         ↑                         ↑
         └─────────────────────────┴────── types
```

### 4.1 `internal/app/context.go`
- 在 `syncContext` 中實現 `ctxCfg.ActiveSkills` 與 `ctxCfg.ActiveGroups` 的有效性過濾。
- 確保 `SyncContext` 調用 `saveCtx()` 時將清洗後的配置持久化。

### 4.2 `internal/cmd/status.go`
- 在遍歷 `ctxCfg.ActiveSkills` 與 `ctxCfg.ActiveGroups` 時收集 `missingSkills` 與 `missingGroups`。
- 在輸出表格後，若存在缺失項則渲染 Warning 警告與修復引導指令。

---

## 5. 測試策略 (TDD Test Plan)

1. **單元測試 (`internal/app/context_test.go`)**：
   - **`TestSyncContext_PrunesInvalidActiveSkillsAndGroups`**：
     - 在 Context 中預先設置不存在於 `skills.json` 的 `ActiveSkills` 以及不存在於 `groups.json` 的 `ActiveGroups`；
     - 執行 `app.SyncContext(targetDir, false, false)`；
     - 驗證磁盤上的 `.skills_manager.json` 中已不再包含無效技能與群組。
2. **CLI 整合測試 (`internal/cmd/status_test.go` / `remove_status_test.go`)**：
   - **`TestStatusCommand_WarningForMissingSkills`**：
     - 設置包含失效技能 ID 的 context；
     - 執行 `statusCmd`；
     - 斷言輸出中包含警告信息及 `skill_manager sync` 提示。
3. **全量回歸測試**：
   - 運行 `go test ./...` 確保 100% 通過。

---

## 6. 開發計劃 (Development Plan)

- [x] 步驟 1: 編寫 `internal/app/context_test.go` 與 `internal/cmd/remove_status_test.go` 測試用例（紅燈階段）。
- [x] 步驟 2: 在 `internal/app/context.go` 的 `syncContext` 中實現無效技能與群組清理，並在 `SyncContext` 中確保持久化。
- [x] 步驟 3: 在 `internal/cmd/status.go` 中實現失效技能/群組的檢測與 Warning 提示。
- [x] 步驟 4: 運行全量測試 `go test ./...` 驗證。
- [x] 步驟 5: 更新設計文檔，補充回顧與總結。

---

## 7. 回顧與總結 (Review & Summary)

本次開發順利完成了 `status` 與 `sync` 命令在面對失效技能時的提示與自癒清理功能：

1. **`status` 命令透明度提升**：
   - 在 [`internal/cmd/status.go`](skill_manager/internal/cmd/status.go) 中增加了對 Context 中無效 `ActiveSkills` 與 `ActiveGroups` 的收集與判斷。
   - 在狀態表格底部明確輸出 Warning 警告，列出失效項目與關聯 Agent，並提示用戶執行 `skill_manager sync`（全局模式下為 `skill_manager sync -g`）進行修復。

2. **`sync` 命令自癒與配置持久化**：
   - 在 [`internal/app/context.go`](skill_manager/internal/app/context.go) 的 `syncContext` 中，對 `ctxCfg.ActiveSkills` 與 `ctxCfg.ActiveGroups` 進行過濾，僅保留存在於當前技能庫中的有效項目。
   - 在 `SyncContext` 中調用 `saveCtx()`，將清理後的 Context 狀態寫回磁盤（`.skills_manager.json` 或 `global.json`）。
   - 配合既有的孤兒軟鏈清理邏輯，徹底清除已失效技能的物理軟鏈與配置殘留。

3. **完整測試保障**：
   - 單元測試 [`internal/app/context_test.go`](skill_manager/internal/app/context_test.go) 驗證了 `SyncContext` 對無效技能與群組的過濾與持久化寫入。
   - CLI 測試 [`internal/cmd/remove_status_test.go`](skill_manager/internal/cmd/remove_status_test.go) 驗證了局部與全局 `status` 命令的警告輸出與修復指令提示。
   - 全量測試套件 `go test ./...` 100% 通過。
