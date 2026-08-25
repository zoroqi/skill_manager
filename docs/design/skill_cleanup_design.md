# 技能一致性與清理機制設計文檔 (Skill Consistency & Cleanup Design)

## 1. 概述 (Overview)

當遠端技能倉庫（Registry）在開源項目端刪除或重命名某些 Skill 時，本地在執行 `skill_manager register update <NAME>` 或 `skill_manager register update --all` 後，已刪除的技能仍然殘留在 `skills.json` 配置文件中。

為了保證 `skills.json` 配置文件中的內容始終準確、最新，不會出現本地或遠端已不存在的技能（Orphaned / Stale Skills），本設計針對技能掃描、倉庫更新、倉庫移除以及全量清理機制進行重構與增強，確保各配置文件（`skills.json`, `groups.json`, context 配置）之間的數據一致性。

---

## 2. 現狀問題與原因分析 (Current Problem & Root Cause Analysis)

### 2.1 `ScanSkills` 現有邏輯缺陷
在目前的 [`internal/app/scan.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/scan.go) 中：
1. 讀取現有 `skills.json` 到 `skillMap`。
2. 調用 `scanner.Scan(dir, registryName)` 掃描目錄。
3. 將掃描到的技能寫入 `skillMap`（僅執行覆蓋或新增）。
4. **問題 1（殘留問題）**：若遠端項目刪除了某個技能 `skill-b`，`scanner.Scan` 不會回傳該技能，但 `skillMap` 仍保留了原先載入的 `skill-b`，最後重新寫入 `skills.json`，導致已刪除技能無法被移除。
5. **問題 2（清空問題）**：若遠端項目刪除了全部技能（`len(skills) == 0`），`ScanSkills` 遇到了 `len(skills) == 0` 立即直接返回，未清除舊有技能記錄。

### 2.2 `cleanupSkills` 現有邏輯缺陷
在 [`internal/app/registry.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/registry.go) 的 `cleanupSkills` 中：
1. 僅透過 `os.Stat(s.Path)` 檢查目錄是否存在，未校驗目錄下是否存在有效的 `SKILL.md`（若目錄存在但 `SKILL.md` 已被移除，仍會被視為有效）。
2. 若倉庫已從 `registries.json` 移除，`LoadBaseConfig()` 無法解析出物理絕對路徑，相對路徑的 `os.Stat` 可能產生誤判。
3. 未聯動清理 `groups.json` 中對無效技能 ID 的引用。

---

## 3. 設計方案與業務邏輯 (Design Specification & Business Logic)

### 3.1 技能掃描同步重構 (`ScanSkills`)
重新定義 `ScanSkills(dir string, registryName string) (int, error)` 的更新語義：
1. **Registry 模式 (`registryName != ""`):**
   - 掃描指定物理目錄 `dir`，獲取最新的技能列表 `scannedSkills`。
   - 遍歷現有的 `cfg.Skills`：
     - 保留所有 `s.RegistryName != registryName` 的技能（其他 Registry 或本地技能）。
     - 移除所有原屬於該 `registryName` 的舊技能。
   - 將 `scannedSkills` 全量追加至 `cfg.Skills` 中。
   - 即使 `scannedSkills` 為空（遠端刪除了全部技能），也能正確清空該 Registry 在 `skills.json` 中的記錄。
2. **非 Registry 模式 (`registryName == ""`):**
   - 將目錄 `dir` 下的現有技能替換為最新掃描結果。
3. **儲存與排序:**
   - 寫入 `skills.json`（由 `store.SaveBaseConfig` 自動按 Registry 與 Name 字典序排序）。

### 3.2 技能庫物理校驗與清理機制 (`cleanupSkills` / `PruneSkills`)
提供嚴謹的無效技能校驗與清理邏輯：
1. **有效性判斷標準**：
   - 若 `s.RegistryName != ""`：
     - 該 `RegistryName` 必須存在於 `registries.json` 中；
     - 其物理目錄下的 `filepath.Join(registryPhysicalPath, s.Path, "SKILL.md")` 必須在硬碟上實際存在且可讀。
   - 若 `s.RegistryName == ""`：
     - `filepath.Join(s.Path, "SKILL.md")` 必須在硬碟上實際存在且可讀。
2. **清理範圍**：
   - **`skills.json`**：剔除所有未通過有效性校驗的技能。
   - **`groups.json`**：遍歷所有技能組，剔除其 `skill_ids` 列表中已在 `skills.json` 中不存在的無效技能 ID。

### 3.3 倉庫生命週期聯動
1. **`RegisterUpdate(name)` / `RegisterUpdateAll()`**：
   - 執行 `git pull` 更新本地代碼。
   - 調用重構後的 `ScanSkills(physicalPath, registry.Name)` 同步最新技能狀態（自動剔除已刪除技能）。
   - 調用 `cleanupSkills()` 確保關聯數據（如 `groups.json`）完整一致。
2. **`RegisterRemove(name, deleteFiles)`**：
   - 從 `registries.json` 中移除該倉庫記錄。
   - 若指定 `deleteFiles`，移動本地目錄至 backup 目錄。
   - 從 `skills.json` 中主動移除所屬 `s.RegistryName == name` 的所有技能，並調用 `cleanupSkills()` 同步清理 `groups.json`。
3. **`RegisterAdd(url, name, customPath)`**：
   - `git clone` 成功後，調用 `ScanSkills` 同步寫入技能庫。

---

## 4. 架構設計與代碼分層 (Architecture & Layering)

遵循項目既有的分層規範：
```
cmd/  ──→  app  ──→  store
 ↑          ↑
 └──────────┴────── types
```

### 4.1 `internal/app/scan.go`
- 重構 `ScanSkills(dir string, registryName string) (int, error)`：
  - 根據 `registryName` 精確替換該註冊源的所有技能記錄。
  - 當掃描結果為 0 時，正常清空該註冊源在 `skills.json` 的記錄並保存。

### 4.2 `internal/app/registry.go`
- 優化 `cleanupSkills() error`：
  - 加載 `registries.json` 與 `skills.json`。
  - 檢查每個技能對應的 `SKILL.md` 文件真實存在性。
  - 同步清理 `groups.json` 中的無效 `skill_ids`。
- 在 `RegisterUpdate` 與 `RegisterRemove` 中整合上述校驗與清理邏輯。

---

## 5. 測試策略 (TDD Test Plan)

1. **單元測試 (`internal/app/scan_test.go`)**：
   - **測試新增與更新**：掃描目錄正確寫入 `skills.json`。
   - **測試刪除同步 (Update with Deleted Skills)**：
     - 初始化包含 `skill-1`, `skill-2` 的倉庫；
     - 模擬遠端刪除 `skill-2` 後重新調用 `ScanSkills`；
     - 斷言 `skills.json` 中僅剩下 `skill-1`，`skill-2` 已被成功刪除。
   - **測試全部技能清空 (Update with All Skills Deleted)**：
     - 模擬遠端刪除目錄下所有技能後重新調用 `ScanSkills`；
     - 斷言 `skills.json` 中該 Registry 的技能列表為空。
2. **單元測試 (`internal/app/registry_test.go`)**：
   - **測試 RegisterRemove**：驗證移除 registry 後，`skills.json` 與 `groups.json` 中的關聯技能與 ID 被正確清理。
   - **測試 cleanupSkills**：手動刪除磁盤上的 `SKILL.md` 後調用清理，驗證無效技能與 group 引用被自動過濾。
3. **全量回歸測試**：
   - 運行 `go test ./...` 確保現有 command, context, status, stats, summary, tui 功能不受影響且全部通過。

---

## 6. 開發計劃 (Development Plan)

- [x] 步驟 1: 編寫 `internal/app/scan_test.go` 與 `internal/app/registry_test.go` 中的測試用例（紅燈階段）。
- [x] 步驟 2: 重構 `internal/app/scan.go`，支持精確覆蓋與刪除不存在的技能。
- [x] 步驟 3: 優化 `internal/app/registry.go` 中的 `cleanupSkills` 及生命週期調用，實現 `SKILL.md` 存在性檢查與 `groups.json` 引用清理。
- [x] 步驟 4: 修復現有測試細節（包括 `summary.go` 的路徑檢查容錯），運行全量測試 `go test ./...` 確保 100% 通過。
- [x] 步驟 5: 更新設計文檔，補充回顧與總結。

---

## 7. 回顧與總結 (Review & Summary)

本次開發順利解決了遠端倉庫刪除技能後本地 `skills.json` 殘留的問題，並強化了全局技能庫與關聯配置的一致性保障：

1. **掃描同步機制完善**：
   - 重構了 [`internal/app/scan.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/scan.go) 中的 `ScanSkills`，改變原先無差別合併策略，改為根據 `registryName` 進行精確全量替換。
   - 支援遠端部分刪除與全部刪除場景，確保 `skills.json` 中的技能記錄與倉庫實際狀態始終一致。

2. **物理存在性校驗與深度清理**：
   - 強化了 [`internal/app/registry.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/registry.go) 中的 `cleanupSkills`，校驗技能物理目錄下的 `SKILL.md` 是否真實存在且所屬 Registry 是否合法。
   - 增加了對 [`groups.json`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/types.go) 中無效 `skill_ids` 引用的聯動清理，防止殘留失效引用導致關聯配置髒污。

3. **全流程自動觸發**：
   - 在 `register update <name>`、`register update --all`、`register remove <name>` 等生命週期中自動執行清理與同步，無須使用者手動干預。

4. **全面測試保障 (TDD)**：
   - 在 [`internal/app/scan_test.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/scan_test.go) 與 [`internal/app/registry_test.go`](file:///Users/zoroqi/work/golang/src/zoroqi/skill_manager/internal/app/registry_test.go) 新增單元測試，完整覆蓋了刪除技能 Rescan、清空技能 Rescan、跨 Registry 隔離、物理文件缺失清理以及倉庫移除清理等邊界場景。
   - 全量測試 `go test ./...` 100% 通過。
