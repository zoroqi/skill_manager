# 配置文件與掃描功能開發計劃 (Plan)

## 1. 任務清單

- [x] **Task 1: 定義 JSON 配置結構體 (internal/config)**
  - 定義 `Skill`, `BaseConfig`, `Group`, `GroupsConfig`, `ContextConfig`, `SummaryConfig` 等結構。
  - `Skill` 結構採用 `Metadata map[string]interface{}` 並整合 `yaml:",inline"` 以存儲動態字段。
- [x] **Task 2: 實作配置管理器 (internal/config/manager.go)**
  - 實作 `Init`, `Save`, `Load` 方法。
  - 支持 `SKILL_MANAGER_CONFIG` 環境變數自定義路徑。
  - 確保 `init` 時自動生成四個基礎配置文件。
- [x] **Task 3: 實作掃描邏輯 (internal/scanner/scanner.go)**
  - 優先解析 `SKILL.md` 的 YAML Frontmatter。
  - 自動將未知字段歸類至 `Metadata` 字典。
  - 採用 `md5(parent_dir/skill_name)` 生成唯一 ID。
- [x] **Task 4: CLI 指令實作 (internal/cmd)**
  - 整合 `init`, `scan`, `list` 命令。
  - 提供整齊的表格化清單輸出。
- [x] **Task 5: 目錄結構優化**
  - 將 `main.go` 調整至項目根目錄。
- [x] **Task 6: 環境變數支援與 Help 更新**
  - 支援 `SKILL_MANAGER_CONFIG` 環境變數。
  - 在 CLI Help 中說明路徑規則與配置邏輯。

## 2. 驗證標準
- [x] `go test ./...` 單元測試全數通過。
- [x] `skill_manager init` 生成完整且正確初始化的 JSON 文件。
- [x] `skill_manager scan` 成功解析 YAML 元數據並存入 `metadata` 字典。
- [x] `skill_manager list` 輸出格式正確。
- [x] 透過環境變數 `SKILL_MANAGER_CONFIG` 能正確引導配置路徑。

## 3. 回顧與總結 (Review & Summary)
### 3.1 完成狀況
本階段任務已圓滿完成，建立了一個靈活且具擴展性的技能管理基礎。
- **技術決策:** 使用 `map[string]interface{}` 與 `yaml:",inline"` 處理非核心字段，極大地提升了對不同平臺 SKILL 規範的兼容性，避免了因規範變動導致的頻繁結構體修改。
- **路徑管理:** 引入 `SKILL_MANAGER_CONFIG` 增強了工具在不同部署環境下的靈活性。

### 3.2 下一步建議
- 實作軟鏈 (Symbolic Link) 維護功能，根據 `context.json` 的定義動態鏈接技能目錄。
- 支援 `list` 指令顯示詳細的元數據 (Metadata) 資訊。
- 實作專案級別的配置初始化與切換邏輯。
