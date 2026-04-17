# Add Skill Feature 開發計劃 (Plan)

## 1. 任務清單

- [ ] **Task 1: 更新設計與計劃文檔**
- [ ] **Task 2: 在 internal/config/ 中增加 Agent 支援的常量與輔助函數**
  - 定義支援的 Agent 列表（如 \`gemini\`, \`claude\`, \`codex\`, \`opencode\`, \`agents\`）。
- [ ] **Task 3: 實作 internal/cmd/add.go**
  - 解析 \`--global\` 與 \`--agent\` 參數。
  - 處理 \`.context.json\` 與全局 context 的載入與保存。
  - 根據 \`--agent\` 與當前目錄創建 \`.<agent>/skills\` 目錄。
  - 使用 \`os.Symlink\` 建立軟鏈。
- [ ] **Task 4: 測試**
  - 編寫單元測試驗證 \`add\` 邏輯與軟鏈創建行為。

## 2. 驗證標準
- [ ] \`skill_manager add test-skill --agent gemini\` 能在當前目錄建立 \`.context.json\` 和 \`.gemini/skills/test-skill\` 軟鏈。
- [ ] \`skill_manager add test-skill --global --agent claude\` 能在全局環境建立軟鏈並更新 global_context。
