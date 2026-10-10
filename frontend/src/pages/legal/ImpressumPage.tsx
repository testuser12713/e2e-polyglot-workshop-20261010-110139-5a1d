export function ImpressumPage() {
  return (
    <section className="page">
      <header className="page__header">
        <h1>Impressum</h1>
        <p className="page__description">Angaben gemäß § 5 DDG.</p>
      </header>

      <section className="legal">
        <h2>Anbieter</h2>
        <dl className="contact-grid">
          <dt>Firma</dt>
          <dd>Kfz-Meisterwerkstatt Vogel</dd>
          <dt>Inhaber</dt>
          <dd>Thomas Vogel</dd>
          <dt>Anschrift</dt>
          <dd>
            Werkstattstraße 12
            <br />
            10365 Berlin
            <br />
            Deutschland
          </dd>
          <dt>Telefon</dt>
          <dd className="mono">+49 30 987654-0</dd>
          <dt>E-Mail</dt>
          <dd>kontakt@werkstatt-vogel.de</dd>
          <dt>Umsatzsteuer-ID</dt>
          <dd className="mono">DE123456789</dd>
        </dl>

        <h2>Verantwortlich für den Inhalt</h2>
        <p>Verantwortlich nach § 18 Abs. 2 MStV: Thomas Vogel, Anschrift wie oben.</p>

        <h2>Registereintrag und Kammer</h2>
        <p>
          Handwerksrolle: Handwerkskammer Berlin. Berufsbezeichnung:
          Kraftfahrzeugtechnikermeister (verliehen in Deutschland).
        </p>

        <h2>Streitschlichtung</h2>
        <p>
          Wir sind nicht verpflichtet und nicht bereit, an einem Streitbeilegungsverfahren vor
          einer Verbraucherschlichtungsstelle teilzunehmen.
        </p>

        <h2>Haftung für Inhalte und Links</h2>
        <p>
          Als Diensteanbieter sind wir für eigene Inhalte auf diesen Seiten verantwortlich. Für
          Inhalte externer Links sind ausschließlich deren Betreiber verantwortlich; zum Zeitpunkt
          der Verlinkung waren keine Rechtsverstöße erkennbar.
        </p>

        <h2>Urheberrecht</h2>
        <p>
          Die auf dieser Seite erstellten Inhalte und Werke unterliegen dem deutschen Urheberrecht.
          Vervielfältigung und Verbreitung bedürfen der schriftlichen Zustimmung des Anbieters.
        </p>

        <p className="muted mt-24">
          Die in dieser Anwendung gezeigten Firmen-, Kunden- und Fahrzeugdaten sind Beispieldaten.
        </p>
      </section>
    </section>
  );
}
