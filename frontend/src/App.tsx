import { Navigate, Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import { ErrorState } from "./components/states";
import { RequireWorkshopAuth } from "./auth/RequireWorkshopAuth";
import { AppointmentPage } from "./pages/customer/AppointmentPage";
import { OrderStatusPage } from "./pages/customer/OrderStatusPage";
import { LoginPage } from "./pages/workshop/LoginPage";
import { OrdersPage } from "./pages/workshop/OrdersPage";
import { OrderDetailPage } from "./pages/workshop/OrderDetailPage";
import { DashboardPage } from "./pages/workshop/DashboardPage";
import { ImpressumPage } from "./pages/legal/ImpressumPage";
import { DatenschutzPage } from "./pages/legal/DatenschutzPage";

function NotFoundPage() {
  return (
    <ErrorState
      title="Seite nicht gefunden"
      message="Die aufgerufene Adresse existiert nicht."
      code="not_found"
    />
  );
}

// The complete route table of the contract. Every screen renders inside the
// shared shell (Layout); the workshop screens sit behind RequireWorkshopAuth,
// which the login ticket fills with the real guard.
export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Navigate to="/termin" replace />} />

        <Route path="/termin" element={<AppointmentPage />} />
        <Route path="/auftragsstatus" element={<OrderStatusPage />} />

        <Route path="/werkstatt" element={<Navigate to="/werkstatt/dashboard" replace />} />
        <Route path="/werkstatt/anmeldung" element={<LoginPage />} />
        <Route
          path="/werkstatt/dashboard"
          element={
            <RequireWorkshopAuth>
              <DashboardPage />
            </RequireWorkshopAuth>
          }
        />
        <Route
          path="/werkstatt/auftraege"
          element={
            <RequireWorkshopAuth>
              <OrdersPage />
            </RequireWorkshopAuth>
          }
        />
        <Route
          path="/werkstatt/auftraege/:id"
          element={
            <RequireWorkshopAuth>
              <OrderDetailPage />
            </RequireWorkshopAuth>
          }
        />

        <Route path="/impressum" element={<ImpressumPage />} />
        <Route path="/datenschutz" element={<DatenschutzPage />} />

        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
