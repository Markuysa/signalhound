import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Sidebar } from './components/Sidebar';
import { EmptyState } from './components/EmptyState';

/* App shell: sidebar + routed content. The three screens are separate tickets; until
   they land, each route shows a placeholder so the shell and navigation are usable and
   testable on their own. */
export function App() {
  return (
    <BrowserRouter>
      <div className="flex h-full flex-col min-[820px]:flex-row">
        <Sidebar />
        <main className="min-w-0 flex-1 overflow-auto">
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Placeholder name="Dashboard" />} />
            <Route path="/feed" element={<Placeholder name="Feed" />} />
            <Route path="/settings" element={<Placeholder name="Settings" />} />
            <Route path="*" element={<Placeholder name="Not found" />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}

function Placeholder({ name }: { name: string }) {
  return (
    <div className="p-6">
      <h1 className="mb-4 font-display text-[20px]">{name}</h1>
      <EmptyState title={`${name} is a separate ticket`} hint="This scaffold provides the shell, tokens, primitives and API client it will build on." />
    </div>
  );
}
