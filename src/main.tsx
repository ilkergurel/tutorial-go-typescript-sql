import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { createBrowserRouter, RouterProvider } from 'react-router-dom';
import { AppLayout } from './app/AppLayout';
import { DashboardPage } from './app/routes/DashboardPage';
import { PathPage } from './app/routes/PathPage';
import { TrackPage } from './app/routes/TrackPage';
import { LessonPage } from './app/routes/LessonPage';
import { NotFoundPage } from './app/routes/NotFoundPage';
import { UnverifiedPage } from './app/routes/UnverifiedPage';
import './index.css';

// URL-first navigation: track / lesson live in the path so every view is
// deep-linkable (ADR-0005). Phase 2 adds ?example= and tab params to /lessons/:id.
const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { path: '/', element: <DashboardPage /> },
      { path: '/paths/:pathId', element: <PathPage /> },
      { path: '/tracks/:trackId', element: <TrackPage /> },
      { path: '/lessons/:lessonId', element: <LessonPage /> },
      { path: '/unverified', element: <UnverifiedPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
]);

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>
);
