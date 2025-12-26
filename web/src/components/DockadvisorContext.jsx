import React, {createContext, useContext, useState} from 'react';

const DockadvisorContext = createContext();

export function DockadvisorProvider({children}) {
  const [score, setScore] = useState(100);
  const [isEmpty, setIsEmpty] = useState(true);
  const [isCalculating, setIsCalculating] = useState(false);
  const [editorReady, setEditorReady] = useState(false);

  return (
    <DockadvisorContext.Provider
      value={{
        score,
        setScore,
        isEmpty,
        setIsEmpty,
        isCalculating,
        setIsCalculating,
        editorReady,
        setEditorReady,
      }}
    >
      {children}
    </DockadvisorContext.Provider>
  );
}

export function useDockadvisor() {
  const context = useContext(DockadvisorContext);
  if (!context) {
    throw new Error('useDockadvisor must be used within DockadvisorProvider');
  }
  return context;
}
