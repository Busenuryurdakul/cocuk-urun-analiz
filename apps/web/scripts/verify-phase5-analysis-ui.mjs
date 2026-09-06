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

console.log("PASS: Phase 5 analysis UI is wired on the product detail page");
