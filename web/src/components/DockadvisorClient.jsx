import {DockadvisorEditor, ScoreGauge} from "./Dockadvisor";
import {useDockadvisor} from "./DockadvisorContext";

export function DockadvisorScoreGauge() {
  const {score, isEmpty, isCalculating, editorReady} = useDockadvisor();

  if (!editorReady) {
    return null;
  }

  return (
    <div className="flex-shrink-0 mr-4">
      <ScoreGauge score={score} isEmpty={isEmpty} isCalculating={isCalculating} />
    </div>
  );
}

export function DockadvisorEditorWrapper() {
  const {
    score,
    setScore,
    isEmpty,
    setIsEmpty,
    isCalculating,
    setIsCalculating,
    editorReady,
    setEditorReady,
  } = useDockadvisor();

  return (
    <DockadvisorEditor
      score={score}
      setScore={setScore}
      isEmpty={isEmpty}
      setIsEmpty={setIsEmpty}
      isCalculating={isCalculating}
      setIsCalculating={setIsCalculating}
      editorReady={editorReady}
      setEditorReady={setEditorReady}
    />
  );
}
