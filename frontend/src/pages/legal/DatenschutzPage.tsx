export function DatenschutzPage() {
  return (
    <section className="page">
      <header className="page__header">
        <h1>Datenschutzerklärung</h1>
        <p className="page__description">
          Informationen zur Verarbeitung personenbezogener Daten im Kundenportal.
        </p>
      </header>

      <section className="legal">
        <h2>Verantwortlicher</h2>
        <dl className="contact-grid">
          <dt>Verantwortlicher</dt>
          <dd>Kfz-Meisterwerkstatt Vogel, Inhaber Thomas Vogel</dd>
          <dt>Anschrift</dt>
          <dd>Werkstattstraße 12, 10365 Berlin, Deutschland</dd>
          <dt>Telefon</dt>
          <dd className="mono">+49 30 987654-0</dd>
          <dt>E-Mail</dt>
          <dd>datenschutz@werkstatt-vogel.de</dd>
        </dl>

        <h2>Zwecke der Verarbeitung</h2>
        <p>
          Wir verarbeiten personenbezogene Daten, um Terminanfragen zu bearbeiten, Aufträge
          auszuführen, Rechnungen zu erstellen und den Auftragsstatus im Kundenportal
          bereitzustellen.
        </p>

        <h2>Rechtsgrundlagen</h2>
        <p>
          Die Verarbeitung erfolgt zur Erfüllung eines Vertrags bzw. zur Durchführung
          vorvertraglicher Maßnahmen (Art. 6 Abs. 1 lit. b DSGVO) sowie zur Erfüllung rechtlicher
          Verpflichtungen, etwa steuer- und handelsrechtlicher Aufbewahrungspflichten (Art. 6 Abs. 1
          lit. c DSGVO).
        </p>

        <h2>Arten der verarbeiteten Daten</h2>
        <ul>
          <li>Kundendaten: Name, E-Mail-Adresse, Telefonnummer</li>
          <li>Fahrzeugdaten: Kennzeichen, Marke, Modell, Kilometerstand</li>
          <li>Auftragsdaten: Auftragsnummer, Wunschtermin, Problembeschreibung, Statusverlauf</li>
          <li>Rechnungsdaten: Positionen, Netto-, Steuer- und Bruttobeträge, Rechnungsnummer</li>
        </ul>

        <h2>Empfänger der Daten</h2>
        <p>
          Eine Weitergabe erfolgt nur, soweit dies zur Vertragserfüllung oder aufgrund gesetzlicher
          Pflichten erforderlich ist, etwa an das Steuerbüro. Die Anwendung wird auf einem Server
          innerhalb der Europäischen Union betrieben; ein Auftragsverarbeitungsvertrag mit dem
          Hoster liegt vor. Eine Übermittlung in Drittländer findet nicht statt.
        </p>

        <h2>Empfänger von Benachrichtigungen</h2>
        <p>
          Rechnungsbenachrichtigungen werden ausschließlich in einem internen Postausgang abgelegt.
          Es wird keine echte E-Mail an Dritte versendet.
        </p>

        <h2>Speicherdauer</h2>
        <p>
          Auftrags- und Rechnungsdaten bewahren wir für die gesetzlichen Aufbewahrungsfristen (bis
          zu zehn Jahre) auf. Anfragen ohne Auftrag löschen wir, sobald sie nicht mehr benötigt
          werden.
        </p>

        <h2>Keine Ressourcen Dritter</h2>
        <p>
          Diese Anwendung lädt keine Schriftarten, Skripte, Bilder oder Analysewerkzeuge aus dem
          Netz. Alle Bestandteile liegen im eigenen Auslieferungsstand und werden vom eigenen Server
          geladen.
        </p>

        <h2>Ihre Rechte</h2>
        <p>
          Sie haben das Recht auf Auskunft, Berichtigung, Löschung, Einschränkung der Verarbeitung,
          Datenübertragbarkeit und Widerspruch. Zudem können Sie sich bei einer
          Datenschutz-Aufsichtsbehörde beschweren.
        </p>

        <p className="muted mt-24">
          Stand: Oktober 2026. Die in dieser Anwendung gezeigten Firmen-, Kunden- und Fahrzeugdaten
          sind Beispieldaten.
        </p>
      </section>
    </section>
  );
}
