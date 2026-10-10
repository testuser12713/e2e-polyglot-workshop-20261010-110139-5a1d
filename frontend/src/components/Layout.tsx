import { Outlet } from "react-router-dom";
import { Nav } from "./Nav";
import { Footer } from "./Footer";

// The shell every screen renders inside: sticky header with navigation, the
// page content, and the footer that carries Impressum and Datenschutz (AC-27).
export function Layout() {
  return (
    <div className="app-shell">
      <Nav />
      <main className="app-main" id="main-content">
        <div className="container">
          <Outlet />
        </div>
      </main>
      <Footer />
    </div>
  );
}
