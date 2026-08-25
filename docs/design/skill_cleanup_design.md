# 技能一致性、清理機制與失效記錄設計文檔 (Skill Consistency, Cleanup & Deleted Tracking Design)

## 1. 概述 (Overview)

當遠端技能倉庫（Registry）在開源項目端刪除或重命名某些 Skill 時，本地在執行 `skill_manager register update <NAME>` 或 `skill_manager register update --all` 後，已刪除的技能容易殘留在 `skills.json` 配置文件中；同時各項目的 Context（`.skills_manager.json` 或 `global.json`）在執行 `status` 時，若技能已刪除，原先只能輸出難以識別的 MD5 Hash ID，缺少明確的技能名稱提示。

為了保證 `skills.json` 配置文件中的內容始終準確最新，並提供清晰的失效提示與自動修復自癒能力，本設計涵蓋：
1. **技能掃描與物理校驗重構**：遠端或本地刪除的技能自動從有效 `skills` 列表中剔除，並清理 `groups.json` 中的無效引用。
2. **`skills.json` 內部維護 `deleted_skills` 列表**：無需新增獨立配置文件，直接在 `BaseConfig` 中保存已被刪除/失效的技能元數據。
3. **`status` 命令友善名稱警示**：當 Context 中存在失效技能時，自動結合 `deleted_skills` 將 Hash ID 映射為友善的「技能名稱 + 所屬倉庫 + ID」格式展示。
4. **`sync` 命令自癒修復**：自動過濾 Context 中的無效技能與群組，清除 Agent 目錄中的孤兒軟鏈，並將乾淨配置持久化。

---

## 2. 數據模型 (Data Model)

### 2.1 基礎配置 (`skills.json`)
修改 [`internal/types.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/types.go) 中的 `BaseConfig`：

```go
type BaseConfig struct {
	Version       string  `json:"version"`
	Skills        []Skill `json:"skills"`
	DeletedSkills []Skill `json:"deleted_skills,omitempty"`
}
```

- **`skills`**：當前在磁盤和倉庫中真實有效的技能列表（按 RegistryName 與 Name 字典序排序）。
- **`deleted_skills`**：歷史上曾存在但當前已被刪除或物理缺失的技能元數據列表（按 RegistryName 與 Name 字典序排序，以 `Skill.ID` 唯一去重）。

### 2.2 上下文配置與群組配置
- **`.skills_manager.json` / `global.json` (`ContextConfig`)**：包含 `active_skills` 與 `active_groups`。
- **`groups.json` (`GroupsConfig`)**：包含技能組定義及關聯的 `skill_ids`。

---

## 3. 業務邏輯與流程設計 (Business Logic)

### 3.1 技能掃描同步重構 (`ScanSkills`)
在 [`internal/app/scan.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/scan.go) 中重新定義更新語義：
1. **Registry 模式 (`registryName != ""`):**
   - 掃描指定物理目錄 `dir`，獲取最新的技能列表 `skills`（建立 `scannedIDs` 集合）。
   - 遍歷現有 `cfg.Skills`：
     - 保留所有 `s.RegistryName != registryName` 的技能；
     - 原屬於該 `registryName` 但未出現在 `scannedIDs` 中的舊技能，標記為 `newlyRemoved`。
   - 即使 `len(skills) == 0`（遠端刪除了全部技能），也能正確清空該 Registry 在 `cfg.Skills` 中的記錄。
2. **非 Registry 模式 (`registryName == ""`):**
   - 將目錄 `dir` 下的現有技能替換為最新掃描結果，不在新掃描結果中的技能標記為 `newlyRemoved`。
3. **`deleted_skills` 轉移與恢復:**
   - 將 `newlyRemoved` 中的技能加入 `cfg.DeletedSkills`。
   - 若新掃描到的 `skills` 此前存在於 `cfg.DeletedSkills` 中（重新加回），自動從 `cfg.DeletedSkills` 中剔除。
   - 調用 `store.SaveBaseConfig(cfg)` 保存並觸發 `cleanupSkills()`。

### 3.2 技能庫物理校驗與清理機制 (`cleanupSkills`)
在 [`internal/app/registry.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/registry.go) 中提供嚴謹的無效技能校驗與清理：
1. **有效性判斷標準**：
   - 若 `s.RegistryName != ""`：該 `RegistryName` 必須存在於 `registries.json` 中，且其物理目錄下的 `filepath.Join(registryPhysicalPath, s.Path, "SKILL.md")` 在硬碟上實際存在。
   - 若 `s.RegistryName == ""`：`filepath.Join(s.Path, "SKILL.md")` 必須在硬碟上實際存在。
2. **清理與轉移範圍**：
   - **`skills.json`**：通過檢查的技能保留在 `cfg.Skills`；未通過檢查的技能轉移至 `cfg.DeletedSkills`。
   - **`groups.json`**：遍歷所有技能組，剔除其 `skill_ids` 列表中已在有效 `skills.json` 中不存在的技能 ID。

### 3.3 倉庫生命週期聯動
1. **`RegisterUpdate(name)` / `RegisterUpdateAll()`**：
   - 執行 `git pull` 更新代碼。
   - 調用 `ScanSkills` 同步最新技能（自動剔除已刪除技能並移至 `deleted_skills`）。
   - 調用 `cleanupSkills()` 確保關聯數據（如 `groups.json`）完整一致。
2. **`RegisterRemove(name, deleteFiles)`**：
   - 從 `registries.json` 移除倉庫，若指定 `deleteFiles` 則將目錄移至 backup。
   - 調用 `cleanupSkills()`，所屬技能自動從 `cfg.Skills` 轉移至 `cfg.DeletedSkills`，並清理 `groups.json`。
3. **`RegisterAdd(url, name, customPath)`**：
   - `git clone` 成功後調用 `ScanSkills` 寫入技能庫。

### 3.4 存儲層持久化 (`internal/store/manager.go`)
- 在 `SaveBaseConfig` 與 `LoadBaseConfig` 中，對 `cfg.Skills` 與 `cfg.DeletedSkills` 統一進行相對路徑轉換（`filepath.Rel`）、物理絕對路徑還原（`filepath.Join`）以及字典序排序。

### 3.5 `status` 命令友善展示增強
在 [`internal/cmd/status.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/cmd/status.go) 中：
1. **失效技能檢測**：
   - 遍歷 `ctxCfg.ActiveSkills`，收集未出現在有效 `skills.json` 中的項目至 `missingSkills`。
   - 遍歷 `ctxCfg.ActiveGroups`，收集未出現在 `groups.json` 中的項目至 `missingGroups`。
2. **名稱解析與告警渲染**：
   - 調用 `application.GetDeletedSkills()` 載入 `deletedMap[id] -> Skill`。
   - 對於 `missingSkills` 中的每個條目：
     - 若在 `deletedMap` 中找到且 `RegistryName != ""`，顯示為：
       `Name (registry: <RegistryName>, id: <ID>)`
     - 若在 `deletedMap` 中找到且 `RegistryName == ""`，顯示為：
       `Name (id: <ID>)`
     - 若未找到，回退顯示原始 `<ID>`。
3. **輸出示例**：
   ```text
   Warning: 1 active skill(s) missing or deleted from skills library:
     - my-skill (registry: org_repo, id: 60ee599f1c6077e0eca1916b56cd4ee4) (agents: gemini)
   Warning: 1 active group(s) missing or deleted from groups library:
     - old-group (agents: gemini)

   Please run `skill_manager sync` to clean up invalid skills.
   ```
   （若為全局模式 `-g`，自動提示 `skill_manager sync -g`）。

### 3.6 `sync` 命令自愈修復
在 [`internal/app/context.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/context.go) 中：
1. **Context 配置清洗**：在 `syncContext` 開頭過濾 `ctxCfg.ActiveSkills` 與 `ctxCfg.ActiveGroups`，僅保留有效項目。
2. **軟鏈與 Gitignore 同步**：失效技能在 Agent 目錄下的軟鏈被視為孤兒軟鏈（Orphan Symlinks）並自動清除，同步更新 `.gitignore`。
3. **持久化寫回**：`SyncContext` 調用 `saveCtx()` 將清洗後的 `ContextConfig` 寫回磁盤，完成自癒。

---

## 4. 架構設計與代碼分層 (Architecture & Layering)

遵循分層規範：
```
cmd/status.go, sync.go  ──→  app/scan.go, registry.go, context.go, skill.go  ──→  store/manager.go
          ↑                                       ↑                                      ↑
          └───────────────────────────────────────┴──────────────────────────────────────┴────── types
```

### 4.1 `internal/types.go`
- 在 `BaseConfig` 中定義 `DeletedSkills []Skill \`json:"deleted_skills,omitempty"\``。

### 4.2 `internal/store/manager.go`
- 在 `SaveBaseConfig` 與 `LoadBaseConfig` 中支援 `DeletedSkills` 相對路徑處理與排序。

### 4.3 `internal/app/scan.go` & `internal/app/registry.go` & `internal/app/skill.go`
- `ScanSkills`：精確覆蓋並轉移/恢復 `DeletedSkills`。
- `cleanupSkills`：物理存在性校驗，轉移無效技能至 `DeletedSkills`，清理 `groups.json`。
- `GetDeletedSkills()`：提供對 `DeletedSkills` 的查詢介面。

### 4.4 `internal/app/context.go`
- `syncContext` / `SyncContext`：清洗 `ctxCfg` 中的無效技能與群組，移除孤兒軟鏈並持久化存盤。

### 4.5 `internal/cmd/status.go`
- 檢測缺失項，結合 `GetDeletedSkills()` 渲染友善告警與修復指令。

---

## 5. 測試策略 (TDD Test Plan)

1. **存儲層測試 (`internal/store/config_test.go`)**：
   - `TestBaseConfig_DeletedSkills`：驗證 `SaveBaseConfig` 與 `LoadBaseConfig` 對 `deleted_skills` 的讀寫、相對路徑處理與排序。
2. **掃描與清理測試 (`internal/app/scan_test.go` & `registry_test.go`)**：
   - `TestScanSkills_UpdateWithDeletedSkills` & `TestScanSkills_TracksDeletedSkills`：驗證刪除技能轉移至 `deleted_skills` 及重新掃描時恢復。
   - `TestCleanupSkills_RemovesOrphanedAndMissingSkills`：驗證物理缺失文件清理及轉入 `deleted_skills`。
   - `TestRegisterRemove_CleansUpSkillsAndGroups`：驗證移除倉庫後技能移至 `deleted_skills` 且 group 引用被清空。
3. **Context 與自癒測試 (`internal/app/context_test.go`)**：
   - `TestSyncContext_PrunesInvalidActiveSkillsAndGroups`：驗證 `SyncContext` 自動清理無效配置並持久化。
4. **CLI 命令測試 (`internal/cmd/remove_status_test.go`)**：
   - `TestStatusCommand_WarningForMissingSkills`：驗證 `status` 與 `status -g` 輸出友善名稱警示與修復引導。
5. **全量回歸測試**：
   - 運行 `go test ./...` 確保 100% 通過。

---

## 6. 開發計劃 (Development Plan)

- [x] 步驟 1: 在 `internal/types.go` 的 `BaseConfig` 中新增 `DeletedSkills`，更新 `internal/store/manager.go` 並編寫單元測試。
- [x] 步驟 2: 重構 `internal/app/scan.go`，支持精確全量替換並自動維護 `DeletedSkills`（刪除轉移與重新加入恢復）。
- [x] 步驟 3: 優化 `internal/app/registry.go` 中的 `cleanupSkills`，實現 `SKILL.md` 存在性檢查、`DeletedSkills` 轉移與 `groups.json` 引用清理。
- [x] 步驟 4: 在 `internal/app/context.go` 中實現 `sync` 對無效 `ActiveSkills` / `ActiveGroups` 的過濾與持久化自癒。
- [x] 步驟 5: 在 `internal/cmd/status.go` 中實現友善名稱格式化輸出與修復引導。
- [x] 步驟 6: 運行全量測試 `go test ./...` 驗證。
- [x] 步驟 7: 更新與合併設計文檔，補充回顧與總結。

---

## 7. 回顧與總結 (Review & Summary)

本次重構與功能增強徹底解決了遠端倉庫刪除技能後本地配置殘留、缺失技能無法友善識別以及 Context 缺少自癒機制的一系列問題：

1. **數據結構精簡優雅**：
   - 無需引入額外的設定檔案，直接在 [`internal/types.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/types.go) 的 `BaseConfig` 內部增加 `deleted_skills` 欄位。
   - 存儲層 [`internal/store/manager.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/store/manager.go) 統一處理相對路徑與字典序排序，確保跨環境一致性。

2. **全流程生命週期一致性**：
   - **掃描同步**：[`internal/app/scan.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/scan.go) 中的 `ScanSkills` 改為精確覆蓋策略，遠端刪除的技能自動移至 `deleted_skills`；若技能被重新加入，則自動移回有效列表。
   - **深度清理**：[`internal/app/registry.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/registry.go) 中的 `cleanupSkills` 嚴格檢查 `SKILL.md` 物理存在性，並聯動清理 `groups.json` 中的失效 ID。

3. **Status 友善性與透明度提升**：
   - [`internal/cmd/status.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/cmd/status.go) 在檢測到 Context 含有失效技能時，自動從 `deleted_skills` 解析原技能名稱與所屬倉庫，提供直觀清晰的 Warning 與引導提示。

4. **Sync 自動修復與自癒**：
   - [`internal/app/context.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/context.go) 中的 `SyncContext` 在同步軟鏈時主動過濾無效配置並持久化寫回磁盤，清除孤兒軟鏈，完成配置自癒。

5. **高覆蓋率測試保障 (TDD)**：
   - 單元測試 [`internal/store/config_test.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/store/config_test.go)、[`internal/app/scan_test.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/scan_test.go)、[`internal/app/registry_test.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/registry_test.go)、[`internal/app/context_test.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/context_test.go) 以及 CLI 測試 [`internal/cmd/remove_status_test.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/cmd/remove_status_test.go) 完整覆蓋所有邊界情況，全量測試 `go test ./...` 100% 通過。
