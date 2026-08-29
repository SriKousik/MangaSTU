import { useState, useEffect } from 'react';
import { Navbar } from './components/Navbar';
import { HomeView } from './components/HomeView';
import { PuzzleView, type PuzzleSubTab } from './components/PuzzleView';
import { AniListCallbackView } from './components/AniListCallbackView';
import { Footer } from './components/Footer';

function App() {
  const isAuthCallback = typeof window !== 'undefined' && window.location.pathname.startsWith('/auth/anilist/callback');
  const [activeTab, setActiveTab] = useState<'home' | 'puzzle'>('home');
  const [puzzleSubTab, setPuzzleSubTab] = useState<PuzzleSubTab>('convert');

  useEffect(() => {
    if (window.location.hash.includes('tracking')) {
      setActiveTab('puzzle');
      setPuzzleSubTab('tracking');
    }
  }, []);

  const handleOpenPuzzle = (subTab?: PuzzleSubTab) => {
    setActiveTab('puzzle');
    if (subTab) {
      setPuzzleSubTab(subTab);
    }
  };

  if (isAuthCallback) {
    return (
      <>
        <Navbar 
          activeTab="puzzle" 
          setActiveTab={setActiveTab}
        />
        <AniListCallbackView onSuccess={() => {
          setPuzzleSubTab('tracking');
        }} />
        <Footer />
      </>
    );
  }

  return (
    <>
      <Navbar 
        activeTab={activeTab} 
        setActiveTab={setActiveTab}
      />

      {activeTab === 'home' ? (
        <HomeView onOpenPuzzle={handleOpenPuzzle} />
      ) : (
        <PuzzleView 
          subTab={puzzleSubTab} 
          setSubTab={setPuzzleSubTab} 
        />
      )}

      <Footer />
    </>
  );
}

export default App;
