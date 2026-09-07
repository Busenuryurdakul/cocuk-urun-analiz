import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const pagePath = join(root, "src/app/org/[id]/products/[productId]/page.tsx");
const panelPath = join(root, "src/components/analysis-panel.tsx");

const page = readFileSync(pagePath, "utf8");
const panel = readFileSync(panelPath, "utf8");

function assert(condition, message) {
  if (!condition) {
    console.error(`FAIL: ${message}`);
    process.exit(1);
  }
}

assert(page.includes('import { AnalysisPanel } from "@/components/analysis-panel"'), "product detail page imports AnalysisPanel");
assert(page.includes("<AnalysisPanel orgId={orgId} productId={productId}"), "product detail page renders AnalysisPanel with org/product ids");
assert(panel.includes("startAgentRun"), "analysis panel calls startAgentRun");
assert(panel.includes("cancelAgentRun"), "analysis panel calls cancelAgentRun");
assert(panel.includes("agentRunEvents"), "analysis panel polls agentRunEvents");
assert(!panel.match(/setInterval\s*\([^)]*=>\s*\{[^}]*setEvents/s), "analysis panel must not fake progress with timer-only event injection");
assert(panel.includes("Import durumundan ayrıdır"), "analysis panel separates import status from analysis status");
assert(panel.includes("finalResult"), "analysis panel reads finalResult");
assert(panel.includes("Politika sonucu"), "analysis panel shows policy-driven decision");
assert(panel.includes("reviewInsights"), "analysis panel shows review insights");
assert(panel.includes("safetyFindings"), "analysis panel shows safety findings");
assert(panel.includes("Geri çağırma eşleşmeleri"), "analysis panel shows recall matches");
assert(!panel.includes("LLM says"), "analysis panel must not attribute BLOCK to the LLM");

console.log("PASS: Phase 5 analysis UI is wired on the product detail page");
