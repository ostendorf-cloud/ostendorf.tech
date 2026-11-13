<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { GovuePage } from '@ostendorf-cloud/govue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const germanHref = computed(
  () => router.resolve({ name: 'privacy', params: { locale: 'de' } }).href,
)
</script>

<template>
  <GovuePage width="reading" :title="t('legal.privacy.title')">
    <div v-if="route.params.locale === 'de'" class="flex flex-col gap-4 text-text">
      <p class="text-text-muted">
        Kurz vorab: Diese Website setzt ein einziges, rein technisches Cookie zum Schutz des
        Kontaktformulars (siehe Abschnitt „Cookies“), aber keine Tracking- oder Werbedienste, und
        bindet keine Inhalte von Drittanbietern ein. Ihr Browser lädt alle Inhalte ausschließlich
        von unserem eigenen Server. Eine Besucherstatistik erstellen wir selbst auf unseren eigenen
        Servern, ohne Skript in Ihrem Browser und ohne sonst etwas auf Ihrem Endgerät zu speichern.
      </p>

      <h2 class="text-xl font-semibold">1. Datenschutz auf einen Blick</h2>

      <h3 class="text-lg font-semibold">Allgemeine Hinweise</h3>
      <p>
        Die folgenden Hinweise geben einen einfachen Überblick darüber, was mit Ihren
        personenbezogenen Daten passiert, wenn Sie diese Website besuchen. Personenbezogene Daten
        sind alle Daten, mit denen Sie persönlich identifiziert werden können. Ausführliche
        Informationen zum Thema Datenschutz entnehmen Sie unserer unter diesem Text aufgeführten
        Datenschutzerklärung.
      </p>

      <h3 class="text-lg font-semibold">Datenerfassung auf dieser Website</h3>

      <h4 class="font-semibold">
        Wer ist verantwortlich für die Datenerfassung auf dieser Website?
      </h4>
      <p>
        Die Datenverarbeitung auf dieser Website erfolgt durch den Websitebetreiber. Dessen
        Kontaktdaten können Sie dem Abschnitt „Hinweis zur Verantwortlichen Stelle“ in dieser
        Datenschutzerklärung entnehmen.
      </p>

      <h4 class="font-semibold">Wie erfassen wir Ihre Daten?</h4>
      <p>
        Ihre Daten werden zum einen dadurch erhoben, dass Sie uns diese mitteilen. Hierbei kann es
        sich z. B. um Daten handeln, die Sie in ein Kontaktformular eingeben.
      </p>
      <p>
        Andere Daten werden automatisch beim Besuch der Website durch unsere IT-Systeme erfasst. Das
        sind vor allem technische Daten (z. B. Internetbrowser, Betriebssystem oder Uhrzeit des
        Seitenaufrufs). Die Erfassung dieser Daten erfolgt automatisch, sobald Sie diese Website
        betreten.
      </p>

      <h4 class="font-semibold">Wofür nutzen wir Ihre Daten?</h4>
      <p>
        Ein Teil der Daten wird erhoben, um eine fehlerfreie Bereitstellung der Website zu
        gewährleisten. Aus den Seitenaufrufen erstellen wir außerdem eine pseudonyme
        Besucherstatistik (siehe „Besucherstatistik mit Umami“). Sofern über die Website Verträge
        geschlossen oder angebahnt werden können, werden die übermittelten Daten auch für
        Vertragsangebote, Bestellungen oder sonstige Auftragsanfragen verarbeitet.
      </p>

      <h4 class="font-semibold">Welche Rechte haben Sie bezüglich Ihrer Daten?</h4>
      <p>
        Sie haben jederzeit das Recht, unentgeltlich Auskunft über Herkunft, Empfänger und Zweck
        Ihrer gespeicherten personenbezogenen Daten zu erhalten. Sie haben außerdem ein Recht, die
        Berichtigung oder Löschung dieser Daten zu verlangen. Wenn Sie eine Einwilligung zur
        Datenverarbeitung erteilt haben, können Sie diese Einwilligung jederzeit für die Zukunft
        widerrufen. Außerdem haben Sie das Recht, unter bestimmten Umständen die Einschränkung der
        Verarbeitung Ihrer personenbezogenen Daten zu verlangen. Des Weiteren steht Ihnen ein
        Beschwerderecht bei der zuständigen Aufsichtsbehörde zu.
      </p>
      <p>
        Hierzu sowie zu weiteren Fragen zum Thema Datenschutz können Sie sich jederzeit an uns
        wenden.
      </p>

      <h3 class="text-lg font-semibold">Analyse-Tools und Tools von Drittanbietern</h3>
      <p>
        Beim Besuch dieser Website werden die aufgerufenen Seiten statistisch ausgewertet. Das
        geschieht mit der Open-Source-Software Umami, die wir selbst auf unseren eigenen Servern
        betreiben; es werden dafür keine Cookies gesetzt, kein Skript in Ihrem Browser ausgeführt
        und keine Daten an Dritte übermittelt. Tracking- oder Werbedienste und Tools von
        Drittanbietern setzen wir nicht ein. Details finden Sie im Abschnitt „Besucherstatistik mit
        Umami“.
      </p>

      <h2 class="text-xl font-semibold">2. Hosting</h2>

      <h3 class="text-lg font-semibold">Hosting auf eigenen Servern</h3>
      <p>
        Wir hosten die Inhalte unserer Website selbst. Die Website läuft auf eigenen Servern, die
        wir in unseren eigenen Räumlichkeiten in Deutschland betreiben. Einen externen
        Hosting-Anbieter setzen wir für die Website nicht ein; die beim Besuch der Website
        anfallenden Daten (siehe „Server-Log-Dateien“) werden auf diesen Servern verarbeitet.
      </p>
      <p>
        Der Betrieb erfolgt auf Grundlage von Art. 6 Abs. 1 lit. f DSGVO. Wir haben ein berechtigtes
        Interesse an einer möglichst zuverlässigen und datensparsamen Darstellung unserer Website.
      </p>

      <h2 class="text-xl font-semibold">3. Allgemeine Hinweise und Pflichtinformationen</h2>

      <h3 class="text-lg font-semibold">Datenschutz</h3>
      <p>
        Die Betreiber dieser Seiten nehmen den Schutz Ihrer persönlichen Daten sehr ernst. Wir
        behandeln Ihre personenbezogenen Daten vertraulich und entsprechend den gesetzlichen
        Datenschutzvorschriften sowie dieser Datenschutzerklärung.
      </p>
      <p>
        Wenn Sie diese Website benutzen, werden verschiedene personenbezogene Daten erhoben.
        Personenbezogene Daten sind Daten, mit denen Sie persönlich identifiziert werden können. Die
        vorliegende Datenschutzerklärung erläutert, welche Daten wir erheben und wofür wir sie
        nutzen. Sie erläutert auch, wie und zu welchem Zweck das geschieht.
      </p>
      <p>
        Wir weisen darauf hin, dass die Datenübertragung im Internet (z. B. bei der Kommunikation
        per E-Mail) Sicherheitslücken aufweisen kann. Ein lückenloser Schutz der Daten vor dem
        Zugriff durch Dritte ist nicht möglich.
      </p>

      <h3 id="responsible" class="text-lg font-semibold">Hinweis zur verantwortlichen Stelle</h3>
      <p>Die verantwortliche Stelle für die Datenverarbeitung auf dieser Website ist:</p>
      <p>
        Simon Ostendorf<br />
        Frankenstr. 8<br />
        52070 Aachen
      </p>
      <p>
        Telefon:
        <a class="text-accent-text hover:underline" href="tel:+4915730971784">+49 1573 0971784</a
        ><br />
        E-Mail:
        <a class="text-accent-text hover:underline" href="mailto:simon@ostendorf.tech"
          >simon@ostendorf.tech</a
        >
      </p>
      <p>
        Verantwortliche Stelle ist die natürliche oder juristische Person, die allein oder gemeinsam
        mit anderen über die Zwecke und Mittel der Verarbeitung von personenbezogenen Daten (z. B.
        Namen, E-Mail-Adressen o. Ä.) entscheidet.
      </p>

      <h3 class="text-lg font-semibold">Speicherdauer</h3>
      <p>
        Soweit innerhalb dieser Datenschutzerklärung keine speziellere Speicherdauer genannt wurde,
        verbleiben Ihre personenbezogenen Daten bei uns, bis der Zweck für die Datenverarbeitung
        entfällt. Wenn Sie ein berechtigtes Löschersuchen geltend machen oder eine Einwilligung zur
        Datenverarbeitung widerrufen, werden Ihre Daten gelöscht, sofern wir keine anderen rechtlich
        zulässigen Gründe für die Speicherung Ihrer personenbezogenen Daten haben (z. B. steuer-
        oder handelsrechtliche Aufbewahrungsfristen); im letztgenannten Fall erfolgt die Löschung
        nach Fortfall dieser Gründe.
      </p>

      <h3 class="text-lg font-semibold">
        Allgemeine Hinweise zu den Rechtsgrundlagen der Datenverarbeitung auf dieser Website
      </h3>
      <p>
        Sofern Sie in die Datenverarbeitung eingewilligt haben, verarbeiten wir Ihre
        personenbezogenen Daten auf Grundlage von Art. 6 Abs. 1 lit. a DSGVO bzw. Art. 9 Abs. 2 lit.
        a DSGVO, sofern besondere Datenkategorien nach Art. 9 Abs. 1 DSGVO verarbeitet werden. Im
        Falle einer ausdrücklichen Einwilligung in die Übertragung personenbezogener Daten in
        Drittstaaten erfolgt die Datenverarbeitung außerdem auf Grundlage von Art. 49 Abs. 1 lit. a
        DSGVO. Sofern Sie in die Speicherung von Cookies oder in den Zugriff auf Informationen in
        Ihr Endgerät (z. B. via Device-Fingerprinting) eingewilligt haben, erfolgt die
        Datenverarbeitung zusätzlich auf Grundlage von § 25 Abs. 1 TDDDG. Die Einwilligung ist
        jederzeit widerrufbar. Sind Ihre Daten zur Vertragserfüllung oder zur Durchführung
        vorvertraglicher Maßnahmen erforderlich, verarbeiten wir Ihre Daten auf Grundlage des Art. 6
        Abs. 1 lit. b DSGVO. Des Weiteren verarbeiten wir Ihre Daten, sofern diese zur Erfüllung
        einer rechtlichen Verpflichtung erforderlich sind auf Grundlage von Art. 6 Abs. 1 lit. c
        DSGVO. Die Datenverarbeitung kann ferner auf Grundlage unseres berechtigten Interesses nach
        Art. 6 Abs. 1 lit. f DSGVO erfolgen. Über die jeweils im Einzelfall einschlägigen
        Rechtsgrundlagen wird in den folgenden Absätzen dieser Datenschutzerklärung informiert.
      </p>

      <h3 class="text-lg font-semibold">Empfänger von personenbezogenen Daten</h3>
      <p>
        Im Rahmen unserer Geschäftstätigkeit arbeiten wir mit verschiedenen externen Stellen
        zusammen. Dabei ist teilweise auch eine Übermittlung von personenbezogenen Daten an diese
        externen Stellen erforderlich. Wir geben personenbezogene Daten nur dann an externe Stellen
        weiter, wenn dies im Rahmen einer Vertragserfüllung erforderlich ist, wenn wir gesetzlich
        hierzu verpflichtet sind (z. B. Weitergabe von Daten an Steuerbehörden), wenn wir ein
        berechtigtes Interesse nach Art. 6 Abs. 1 lit. f DSGVO an der Weitergabe haben oder wenn
        eine sonstige Rechtsgrundlage die Datenweitergabe erlaubt. Beim Einsatz von
        Auftragsverarbeitern geben wir personenbezogene Daten unserer Kunden nur auf Grundlage eines
        gültigen Vertrags über Auftragsverarbeitung weiter. Im Falle einer gemeinsamen Verarbeitung
        wird ein Vertrag über gemeinsame Verarbeitung geschlossen.
      </p>

      <h3 class="text-lg font-semibold">Widerruf Ihrer Einwilligung zur Datenverarbeitung</h3>
      <p>
        Viele Datenverarbeitungsvorgänge sind nur mit Ihrer ausdrücklichen Einwilligung möglich. Sie
        können eine bereits erteilte Einwilligung jederzeit widerrufen. Die Rechtmäßigkeit der bis
        zum Widerruf erfolgten Datenverarbeitung bleibt vom Widerruf unberührt.
      </p>

      <h3 class="text-lg font-semibold">
        Widerspruchsrecht gegen die Datenerhebung in besonderen Fällen sowie gegen Direktwerbung
        (Art. 21 DSGVO)
      </h3>
      <p class="uppercase">
        Wenn die Datenverarbeitung auf Grundlage von Art. 6 Abs. 1 lit. e oder f DSGVO erfolgt,
        haben Sie jederzeit das Recht, aus Gründen, die sich aus Ihrer besonderen Situation ergeben,
        gegen die Verarbeitung Ihrer personenbezogenen Daten Widerspruch einzulegen; dies gilt auch
        für ein auf diese Bestimmungen gestütztes Profiling. Die jeweilige Rechtsgrundlage, auf
        denen eine Verarbeitung beruht, entnehmen Sie dieser Datenschutzerklärung. Wenn Sie
        Widerspruch einlegen, werden wir Ihre betroffenen personenbezogenen Daten nicht mehr
        verarbeiten, es sei denn, wir können zwingende schutzwürdige Gründe für die Verarbeitung
        nachweisen, die Ihre Interessen, Rechte und Freiheiten überwiegen oder die Verarbeitung
        dient der Geltendmachung, Ausübung oder Verteidigung von Rechtsansprüchen (Widerspruch nach
        Art. 21 Abs. 1 DSGVO).
      </p>
      <p class="uppercase">
        Werden Ihre personenbezogenen Daten verarbeitet, um Direktwerbung zu betreiben, so haben Sie
        das Recht, jederzeit Widerspruch gegen die Verarbeitung Sie betreffender personenbezogenen
        Daten zum Zwecke derartiger Werbung einzulegen; dies gilt auch für das Profiling, soweit es
        mit solcher Direktwerbung in Verbindung steht. Wenn Sie widersprechen, werden Ihre
        personenbezogenen Daten anschließend nicht mehr zum Zwecke der Direktwerbung verwendet
        (Widerspruch nach Art. 21 Abs. 2 DSGVO).
      </p>

      <h3 class="text-lg font-semibold">Beschwerderecht bei der zuständigen Aufsichtsbehörde</h3>
      <p>
        Im Falle von Verstößen gegen die DSGVO steht den Betroffenen ein Beschwerderecht bei einer
        Aufsichtsbehörde, insbesondere in dem Mitgliedstaat ihres gewöhnlichen Aufenthalts, ihres
        Arbeitsplatzes oder des Orts des mutmaßlichen Verstoßes zu. Das Beschwerderecht besteht
        unbeschadet anderweitiger verwaltungsrechtlicher oder gerichtlicher Rechtsbehelfe.
      </p>

      <h3 class="text-lg font-semibold">Recht auf Datenübertragbarkeit</h3>
      <p>
        Sie haben das Recht, Daten, die wir auf Grundlage Ihrer Einwilligung oder in Erfüllung eines
        Vertrags automatisiert verarbeiten, an sich oder an einen Dritten in einem gängigen,
        maschinenlesbaren Format aushändigen zu lassen. Sofern Sie die direkte Übertragung der Daten
        an einen anderen Verantwortlichen verlangen, erfolgt dies nur, soweit es technisch machbar
        ist.
      </p>

      <h3 class="text-lg font-semibold">Auskunft, Berichtigung und Löschung</h3>
      <p>
        Sie haben im Rahmen der geltenden gesetzlichen Bestimmungen jederzeit das Recht auf
        unentgeltliche Auskunft über Ihre gespeicherten personenbezogenen Daten, deren Herkunft und
        Empfänger und den Zweck der Datenverarbeitung und ggf. ein Recht auf Berichtigung oder
        Löschung dieser Daten. Hierzu sowie zu weiteren Fragen zum Thema personenbezogene Daten
        können Sie sich jederzeit an uns wenden.
      </p>

      <h3 class="text-lg font-semibold">Recht auf Einschränkung der Verarbeitung</h3>
      <p>
        Sie haben das Recht, die Einschränkung der Verarbeitung Ihrer personenbezogenen Daten zu
        verlangen. Hierzu können Sie sich jederzeit an uns wenden. Das Recht auf Einschränkung der
        Verarbeitung besteht in folgenden Fällen:
      </p>
      <ul class="list-disc pl-6">
        <li>
          Wenn Sie die Richtigkeit Ihrer bei uns gespeicherten personenbezogenen Daten bestreiten,
          benötigen wir in der Regel Zeit, um dies zu überprüfen. Für die Dauer der Prüfung haben
          Sie das Recht, die Einschränkung der Verarbeitung Ihrer personenbezogenen Daten zu
          verlangen.
        </li>
        <li>
          Wenn die Verarbeitung Ihrer personenbezogenen Daten unrechtmäßig geschah/geschieht, können
          Sie statt der Löschung die Einschränkung der Datenverarbeitung verlangen.
        </li>
        <li>
          Wenn wir Ihre personenbezogenen Daten nicht mehr benötigen, Sie sie jedoch zur Ausübung,
          Verteidigung oder Geltendmachung von Rechtsansprüchen benötigen, haben Sie das Recht,
          statt der Löschung die Einschränkung der Verarbeitung Ihrer personenbezogenen Daten zu
          verlangen.
        </li>
        <li>
          Wenn Sie einen Widerspruch nach Art. 21 Abs. 1 DSGVO eingelegt haben, muss eine Abwägung
          zwischen Ihren und unseren Interessen vorgenommen werden. Solange noch nicht feststeht,
          wessen Interessen überwiegen, haben Sie das Recht, die Einschränkung der Verarbeitung
          Ihrer personenbezogenen Daten zu verlangen.
        </li>
      </ul>
      <p>
        Wenn Sie die Verarbeitung Ihrer personenbezogenen Daten eingeschränkt haben, dürfen diese
        Daten – von ihrer Speicherung abgesehen – nur mit Ihrer Einwilligung oder zur
        Geltendmachung, Ausübung oder Verteidigung von Rechtsansprüchen oder zum Schutz der Rechte
        einer anderen natürlichen oder juristischen Person oder aus Gründen eines wichtigen
        öffentlichen Interesses der Europäischen Union oder eines Mitgliedstaats verarbeitet werden.
      </p>

      <h3 class="text-lg font-semibold">SSL- bzw. TLS-Verschlüsselung</h3>
      <p>
        Diese Seite nutzt aus Sicherheitsgründen und zum Schutz der Übertragung vertraulicher
        Inhalte, wie zum Beispiel Bestellungen oder Anfragen, die Sie an uns als Seitenbetreiber
        senden, eine SSL- bzw. TLS-Verschlüsselung. Eine verschlüsselte Verbindung erkennen Sie
        daran, dass die Adresszeile des Browsers von „http://“ auf „https://“ wechselt und an dem
        Schloss-Symbol in Ihrer Browserzeile.
      </p>
      <p>
        Wenn die SSL- bzw. TLS-Verschlüsselung aktiviert ist, können die Daten, die Sie an uns
        übermitteln, nicht von Dritten mitgelesen werden.
      </p>

      <h3 class="text-lg font-semibold">Widerspruch gegen Werbe-E-Mails</h3>
      <p>
        Der Nutzung von im Rahmen der Impressumspflicht veröffentlichten Kontaktdaten zur
        Übersendung von nicht ausdrücklich angeforderter Werbung und Informationsmaterialien wird
        hiermit widersprochen. Die Betreiber der Seiten behalten sich ausdrücklich rechtliche
        Schritte im Falle der unverlangten Zusendung von Werbeinformationen, etwa durch
        Spam-E-Mails, vor.
      </p>

      <h2 class="text-xl font-semibold">4. Datenerfassung auf dieser Website</h2>

      <h3 class="text-lg font-semibold">Cookies</h3>
      <p>
        Diese Website setzt ein einziges Cookie: ein technisches Sitzungs-Cookie, das bei jedem
        Seitenaufruf gesetzt wird, sofern Ihr Browser noch keines hat. Es enthält ausschließlich
        eine zufällige, nicht auf Sie persönlich zurückführbare Kennung und dient dazu, unserem
        Server zu ermöglichen, das Kontaktformular vor Cross-Site-Request-Forgery-Angriffen (CSRF)
        zu schützen; es wird nicht zu Tracking, Analyse oder Werbung verwendet und nicht an Dritte
        übermittelt.
      </p>
      <p>
        Das Cookie ist ein Sitzungs-Cookie ohne eigenes Speicherdauer-Attribut: Ihr Browser verwirft
        es, sobald Sie ihn schließen. Zusätzlich ist es so gesetzt, dass kein Skript auf der Seite
        es auslesen kann (HttpOnly), dass es nur über eine verschlüsselte Verbindung übertragen wird
        (Secure) und dass es nicht an eine andere Website mitgesendet wird (SameSite=Lax).
      </p>
      <p>
        Nach unserer Einschätzung ist dieses Cookie technisch notwendig im Sinne von § 25 Abs. 2 Nr.
        2 TDDDG, weil ohne ein solches Sitzungs-Cookie das Kontaktformular nicht wirksam vor
        Cross-Site-Request-Forgery geschützt werden kann; eine Einwilligung ist deshalb nicht
        erforderlich, und wir zeigen kein Cookie-Banner an. Darüber hinaus speichern wir keine
        weiteren Informationen in Ihrem Endgerät, greifen nicht auf dort gespeicherte Informationen
        zu und setzen auch kein Device-Fingerprinting im Browser ein.
      </p>

      <h3 class="text-lg font-semibold">Server-Log-Dateien</h3>
      <p>
        Bei jedem Aufruf dieser Website schreibt unser Webserver automatisch eine Protokollzeile.
        Darin festgehalten werden:
      </p>
      <ul class="list-disc pl-6">
        <li>Zeitpunkt der Anfrage</li>
        <li>IP-Adresse</li>
        <li>
          eine zufällige, von unserem Server vergebene Anfrage-Kennung, mit der sich
          zusammengehörige Protokollzeilen einer einzelnen Anfrage zuordnen lassen
        </li>
        <li>Anfragemethode (z. B. GET oder POST)</li>
        <li>
          eine grobe, interne Einordnung der Anfrage (z. B. „eine Seite dieser Website“, „eine
          Asset-Datei“), nicht die vollständige aufgerufene Adresse mit allen enthaltenen
          Parametern; bei einem gewöhnlichen Seitenaufruf unterscheidet diese Einordnung nicht
          einmal, welche einzelne Seite aufgerufen wurde – Startseite, Impressum,
          Datenschutzerklärung, AGB und die Fehlerseite erhalten alle denselben Wert
        </li>
        <li>HTTP-Statuscode und übertragene Datenmenge</li>
        <li>Bearbeitungsdauer der Anfrage</li>
      </ul>
      <p>
        Nicht erfasst werden dabei der Browsertyp Ihres Geräts (User-Agent) und eine etwaige
        Referrer-URL.
      </p>
      <p>Eine Zusammenführung dieser Daten mit anderen Datenquellen wird nicht vorgenommen.</p>
      <p>
        Die Erfassung dieser Daten erfolgt auf Grundlage von Art. 6 Abs. 1 lit. f DSGVO. Der
        Websitebetreiber hat ein berechtigtes Interesse an der technisch fehlerfreien Darstellung,
        der Sicherheit und der Optimierung seiner Website sowie an der Abwehr von Missbrauch –
        hierzu müssen die Server-Log-Files erfasst werden.
      </p>
      <p>
        Die Log-Dateien werden nur für einen begrenzten Zeitraum aufbewahrt und spätestens nach 30
        Tagen automatisch gelöscht. Log-Dateien, deren weitere Aufbewahrung zu Beweiszwecken
        erforderlich ist, sind bis zur endgültigen Klärung des jeweiligen Vorfalls von der Löschung
        ausgenommen.
      </p>

      <h3 class="text-lg font-semibold">Ratenbegrenzung</h3>
      <p>
        Um unsere Website vor Überlastung und Missbrauch zu schützen, begrenzen wir, wie viele
        Anfragen eine einzelne IP-Adresse innerhalb kurzer Zeit an uns stellen kann. Dazu zählt
        unser Server für jede Adresse, wie viele Anfragen kurz zuvor eingegangen sind; stellt eine
        Adresse zu viele Anfragen, werden weitere Anfragen von ihr kurzzeitig abgewiesen.
      </p>
      <p>
        Dieser Zähler existiert ausschließlich im Arbeitsspeicher unseres Servers. Er wird nicht auf
        Festplatte geschrieben und nicht an Dritte übermittelt. Sobald eine Adresse eine Zeit lang
        keine weiteren Anfragen mehr stellt, verwirft unser Server den zugehörigen Zähler
        automatisch; eine darüberhinausgehende Aufbewahrung findet nicht statt.
      </p>
      <p>
        Diese allgemeine Ratenbegrenzung gilt für jeden Aufruf unserer Website und ist unabhängig
        von der im Abschnitt „Kontaktformular“ beschriebenen Begrenzung für das Kontaktformular;
        beide werden getrennt gezählt.
      </p>
      <p>
        Die Verarbeitung erfolgt auf Grundlage von Art. 6 Abs. 1 lit. f DSGVO. Wir haben ein
        berechtigtes Interesse daran, unsere Website vor Überlastung und missbräuchlichen Zugriffen
        zu schützen.
      </p>

      <h3 id="umami" class="text-lg font-semibold">Besucherstatistik mit Umami</h3>
      <p>
        Um zu verstehen, welche Inhalte unserer Website gelesen werden und wie Besucher zu uns
        finden, erstellen wir eine Besucherstatistik mit der Open-Source-Software Umami. Umami
        betreiben wir selbst auf unseren eigenen Servern in Deutschland (siehe „Hosting auf eigenen
        Servern“). Sicherungskopien der Statistik-Datenbank speichern wir bei der Hetzner Online
        GmbH, Industriestr. 25, 91710 Gunzenhausen, in einem Rechenzentrum in Deutschland; darüber
        hinaus werden die Daten nicht an Dritte übermittelt.
      </p>
      <p>
        Umami läuft nicht in Ihrem Browser: Es wird kein Skript geladen, kein Cookie gesetzt und
        nichts auf Ihrem Endgerät gespeichert oder ausgelesen. Stattdessen meldet unser Webserver
        jeden Aufruf einer Seite an Umami, mit den Angaben, die Ihr Browser beim Aufruf ohnehin an
        uns übermittelt:
      </p>
      <ul class="list-disc pl-6">
        <li>aufgerufene Seite (ohne angehängte Parameter) und ihr Titel</li>
        <li>
          Referrer URL ohne Parameter, sofern Sie von einer anderen Website kommen und Ihr Browser
          sie übermittelt
        </li>
        <li>bevorzugte Sprache Ihres Browsers</li>
        <li>Browsertyp und Browserversion sowie verwendetes Betriebssystem (User-Agent)</li>
        <li>IP-Adresse</li>
      </ul>
      <p>
        Aus der IP-Adresse ermittelt Umami Land, Region und Stadt. Aus IP-Adresse, User-Agent, einem
        geheimen Schlüssel und dem laufenden Monat bildet Umami eine pseudonyme Kennung, um die
        Seitenaufrufe eines Besuchs zusammenzufassen; die Kennung wechselt deshalb jeden Monat. Die
        IP-Adresse und den vollständigen User-Agent speichert Umami nicht, sondern nur die daraus
        abgeleiteten Angaben (Land, Region, Stadt, Browser, Betriebssystem) zusammen mit der
        pseudonymen Kennung. Eine Zusammenführung mit anderen Daten oder eine Identifizierung
        einzelner Besucher findet nicht statt.
      </p>
      <p>
        Die Statistikdaten werden nach 14 Monaten automatisch gelöscht. In den Sicherungskopien der
        Datenbank bleiben sie danach in der Regel noch bis zu 15 Tage erhalten.
      </p>
      <p>
        Nicht erfasst werden Aufrufe, bei denen Ihr Browser das Signal „Do Not Track“ (DNT) oder
        „Global Privacy Control“ (GPC) sendet, sowie Aufrufe, die Ihr Browser nur vorab lädt.
      </p>
      <p>
        Die Verarbeitung erfolgt auf Grundlage von Art. 6 Abs. 1 lit. f DSGVO. Wir haben ein
        berechtigtes Interesse daran, die Nutzung unserer Website statistisch auszuwerten, um unser
        Angebot zu verbessern, ohne dabei Daten unserer Besucher an Dritte weiterzugeben. Sie können
        der Verarbeitung jederzeit widersprechen, indem Sie in Ihrem Browser „Do Not Track“ oder
        „Global Privacy Control“ aktivieren, oder indem Sie uns kontaktieren.
      </p>

      <h3 class="text-lg font-semibold">Kontaktformular</h3>
      <p>
        Wenn Sie uns per Kontaktformular Anfragen zukommen lassen, werden Ihre Angaben aus dem
        Anfrageformular inklusive der von Ihnen dort angegebenen Kontaktdaten (Name, E-Mail-Adresse
        und Nachricht) zwecks Bearbeitung der Anfrage und für den Fall von Anschlussfragen bei uns
        gespeichert. Diese Daten geben wir nicht ohne Ihre Einwilligung weiter.
      </p>
      <p>
        Ihre Angaben werden von unserem Webserver nicht in einer Datenbank gespeichert, sondern als
        E-Mail über unseren eigenen Mailserver an unser Postfach gesendet. Den Mailserver betreiben
        wir selbst auf einem von uns gemieteten Server der Hetzner Online GmbH, Industriestr. 25,
        91710 Gunzenhausen, in einem Rechenzentrum in Deutschland. Die Übertragung vom Webserver zum
        Mailserver erfolgt verschlüsselt.
      </p>
      <p>
        Zum Schutz vor Spam löst Ihr Browser beim Ausfüllen des Formulars eine kleine Rechenaufgabe
        (sogenannter Proof of Work). Dazu ruft er bei unserem Server eine zufällige Aufgabe ab und
        sendet die Lösung zusammen mit dem Formular zurück. Dabei werden selbst keine weiteren
        Cookies gesetzt, keine weiteren Informationen in Ihrem Endgerät gespeichert und keine
        Dienste von Drittanbietern eingebunden; das technische Sitzungs-Cookie, das beim Aufruf
        jeder Seite dieser Website gesetzt wird (siehe Abschnitt „Cookies“), schützt dabei auch
        dieses Formular vor Cross-Site-Request-Forgery. Um Missbrauch zu verhindern, verarbeitet
        unser Server außerdem Ihre IP-Adresse kurzzeitig im Arbeitsspeicher, um die Zahl der
        Anfragen je Adresse zu begrenzen; dafür wird sie nicht dauerhaft gespeichert.
        Rechtsgrundlage hierfür ist unser berechtigtes Interesse am Schutz vor Spam und Missbrauch
        (Art. 6 Abs. 1 lit. f DSGVO).
      </p>
      <p>
        Die Verarbeitung dieser Daten erfolgt auf Grundlage von Art. 6 Abs. 1 lit. b DSGVO, sofern
        Ihre Anfrage mit der Erfüllung eines Vertrags zusammenhängt oder zur Durchführung
        vorvertraglicher Maßnahmen erforderlich ist. In allen übrigen Fällen beruht die Verarbeitung
        auf unserem berechtigten Interesse an der effektiven Bearbeitung der an uns gerichteten
        Anfragen (Art. 6 Abs. 1 lit. f DSGVO) oder auf Ihrer Einwilligung (Art. 6 Abs. 1 lit. a
        DSGVO) sofern diese abgefragt wurde; die Einwilligung ist jederzeit widerrufbar.
      </p>
      <p>
        Die von Ihnen im Kontaktformular eingegebenen Daten verbleiben bei uns, bis Sie uns zur
        Löschung auffordern, Ihre Einwilligung zur Speicherung widerrufen oder der Zweck für die
        Datenspeicherung entfällt (z. B. nach abgeschlossener Bearbeitung Ihrer Anfrage). Zwingende
        gesetzliche Bestimmungen – insbesondere Aufbewahrungsfristen – bleiben unberührt.
      </p>

      <h3 class="text-lg font-semibold">Anfrage per E-Mail, Telefon oder Telefax</h3>
      <p>
        Wenn Sie uns per E-Mail, Telefon oder Telefax kontaktieren, wird Ihre Anfrage inklusive
        aller daraus hervorgehenden personenbezogenen Daten (Name, Anfrage) zum Zwecke der
        Bearbeitung Ihres Anliegens bei uns gespeichert und verarbeitet. Diese Daten geben wir nicht
        ohne Ihre Einwilligung weiter.
      </p>
      <p>
        Die Verarbeitung dieser Daten erfolgt auf Grundlage von Art. 6 Abs. 1 lit. b DSGVO, sofern
        Ihre Anfrage mit der Erfüllung eines Vertrags zusammenhängt oder zur Durchführung
        vorvertraglicher Maßnahmen erforderlich ist. In allen übrigen Fällen beruht die Verarbeitung
        auf unserem berechtigten Interesse an der effektiven Bearbeitung der an uns gerichteten
        Anfragen (Art. 6 Abs. 1 lit. f DSGVO) oder auf Ihrer Einwilligung (Art. 6 Abs. 1 lit. a
        DSGVO) sofern diese abgefragt wurde; die Einwilligung ist jederzeit widerrufbar.
      </p>
      <p>
        Die von Ihnen an uns per Kontaktanfragen übersandten Daten verbleiben bei uns, bis Sie uns
        zur Löschung auffordern, Ihre Einwilligung zur Speicherung widerrufen oder der Zweck für die
        Datenspeicherung entfällt (z. B. nach abgeschlossener Bearbeitung Ihres Anliegens).
        Zwingende gesetzliche Bestimmungen – insbesondere gesetzliche Aufbewahrungsfristen – bleiben
        unberührt.
      </p>

      <h2 class="text-xl font-semibold">5. Soziale Medien</h2>

      <h3 class="text-lg font-semibold">Instagram-Beiträge auf dieser Website</h3>
      <p>
        Im Abschnitt „Impressionen“ zeigen wir die neuesten Beiträge unseres Instagram-Kontos
        @simonostendorf.vtech. Instagram wird angeboten von der Meta Platforms Ireland Limited,
        Merrion Road, Dublin 4, D04 X2K5, Irland.
      </p>
      <p>
        Die Beiträge (Bilder, Bildbeschreibungen und Links) ruft unser Server regelmäßig über die
        Programmierschnittstelle (API) von Instagram ab, hält sie auf unserem Server vor und liefert
        sie von unserer eigenen Domain aus. Ihr Browser baut dabei keine Verbindung zu Servern von
        Instagram oder Meta auf, und es werden keine Daten über Ihren Besuch an Instagram oder Meta
        übermittelt. Bei diesem Abruf übermittelt unser Server lediglich seine eigene IP-Adresse und
        unseren Zugangsschlüssel an Instagram.
      </p>
      <p>
        Erst wenn Sie einen Beitrag anklicken, öffnet sich dieser auf der Website von Instagram. Ab
        dann gilt die Datenschutzerklärung von Instagram:
        <a
          class="text-accent-text hover:underline"
          href="https://privacycenter.instagram.com/policy/"
          target="_blank"
          rel="noopener noreferrer"
          >https://privacycenter.instagram.com/policy/</a
        >.
      </p>
      <p>
        Die Darstellung der Beiträge erfolgt auf Grundlage von Art. 6 Abs. 1 lit. f DSGVO. Wir haben
        ein berechtigtes Interesse daran, unsere Arbeit anschaulich zu zeigen, ohne dabei Daten
        unserer Besucher an Dritte zu übermitteln.
      </p>

      <h3 class="text-lg font-semibold">Links zu Instagram, Facebook und WhatsApp</h3>
      <p>
        Auf dieser Website verlinken wir auf unsere Profile bei Instagram und Facebook sowie auf
        WhatsApp. Anbieter von Instagram und Facebook ist die Meta Platforms Ireland Limited,
        Merrion Road, Dublin 4, D04 X2K5, Irland, Anbieter von WhatsApp ist die WhatsApp Ireland
        Limited, Merrion Road, Dublin 4, D04 X2K5, Irland.
      </p>
      <p>
        Es handelt sich um einfache Links. Plugins, „Like“-Buttons oder andere Elemente dieser
        Dienste sind auf unserer Website nicht eingebunden, sodass beim Besuch unserer Website keine
        Daten an diese Anbieter übermittelt werden. Erst wenn Sie einen Link anklicken, werden Sie
        auf die Seite des jeweiligen Anbieters weitergeleitet; ab dann gelten dessen
        Datenschutzbestimmungen:
        <a
          class="text-accent-text hover:underline"
          href="https://privacycenter.instagram.com/policy/"
          target="_blank"
          rel="noopener noreferrer"
          >Instagram</a
        >,
        <a
          class="text-accent-text hover:underline"
          href="https://www.facebook.com/privacy/policy/"
          target="_blank"
          rel="noopener noreferrer"
          >Facebook</a
        >
        und
        <a
          class="text-accent-text hover:underline"
          href="https://www.whatsapp.com/legal/privacy-policy-eea"
          target="_blank"
          rel="noopener noreferrer"
          >WhatsApp</a
        >.
      </p>
      <p>
        Wenn Sie uns über WhatsApp schreiben, verarbeiten wir Ihre Telefonnummer und Ihre Nachricht
        wie im Abschnitt „Anfrage per E-Mail, Telefon oder Telefax“ beschrieben. Die Nachricht wird
        dabei über die Server von WhatsApp übermittelt; hierfür gelten die Datenschutzbestimmungen
        von WhatsApp.
      </p>
    </div>

    <div v-else class="flex flex-col gap-4 text-text">
      <p class="text-text-muted">
        {{ t('legal.bindingNotice') }}
        <a class="text-accent-text hover:underline" :href="germanHref">{{
          t('legal.viewGerman')
        }}</a>
      </p>

      <p class="text-text-muted">
        A brief summary up front: this website sets a single, purely technical cookie to protect the
        contact form (see the "Cookies" section), but no tracking or advertising services, and does
        not embed third-party content. Your browser loads everything exclusively from our own
        server. We compile visitor statistics ourselves, on our own servers, without any script
        running in your browser and without storing anything else on your device.
      </p>

      <h2 class="text-xl font-semibold">1. Data protection at a glance</h2>

      <h3 class="text-lg font-semibold">General notes</h3>
      <p>
        The following notes give a simple overview of what happens to your personal data when you
        visit this website. Personal data is any data that can be used to identify you personally.
        For detailed information on data protection, please see our privacy policy set out below
        this text.
      </p>

      <h3 class="text-lg font-semibold">Data collection on this website</h3>

      <h4 class="font-semibold">Who is responsible for data collection on this website?</h4>
      <p>
        Data processing on this website is carried out by the website operator. You can find the
        operator's contact details in the section "Information about the controller" in this privacy
        policy.
      </p>

      <h4 class="font-semibold">How do we collect your data?</h4>
      <p>
        Your data is collected, firstly, through information you provide to us. This may, for
        example, be data you enter into a contact form.
      </p>
      <p>
        Other data is collected automatically by our IT systems when you visit the website. This is
        mainly technical data (for example, your internet browser, operating system or the time of
        the page request). This data is collected automatically as soon as you access this website.
      </p>

      <h4 class="font-semibold">What do we use your data for?</h4>
      <p>
        Part of the data is collected to ensure the website is provided without errors. We also use
        page views to compile pseudonymised visitor statistics (see "Visitor statistics with
        Umami"). To the extent contracts can be concluded or initiated through the website, the data
        transmitted is also processed for contract offers, orders or other service enquiries.
      </p>

      <h4 class="font-semibold">What rights do you have regarding your data?</h4>
      <p>
        You have the right at any time to obtain information, free of charge, about the origin,
        recipients and purpose of your stored personal data. You also have the right to demand that
        this data be corrected or deleted. If you have given consent to data processing, you can
        withdraw that consent at any time for the future. You also have the right, under certain
        circumstances, to demand that the processing of your personal data be restricted.
        Furthermore, you have the right to lodge a complaint with the competent supervisory
        authority.
      </p>
      <p>
        You are welcome to contact us at any time regarding this and any further questions about
        data protection.
      </p>

      <h3 class="text-lg font-semibold">Analytics tools and third-party tools</h3>
      <p>
        When you visit this website, the pages accessed are evaluated statistically. This is done
        using the open-source software Umami, which we operate ourselves on our own servers; no
        cookies are set for this purpose, no script runs in your browser, and no data is transmitted
        to third parties. We do not use tracking or advertising services, or tools from third-party
        providers. Details can be found in the section "Visitor statistics with Umami".
      </p>

      <h2 class="text-xl font-semibold">2. Hosting</h2>

      <h3 class="text-lg font-semibold">Hosting on our own servers</h3>
      <p>
        We host the content of our website ourselves. The website runs on our own servers, which we
        operate on our own premises in Germany. We do not use an external hosting provider for the
        website; the data generated when the website is visited (see "Server log files") is
        processed on these servers.
      </p>
      <p>
        This is carried out on the basis of Art. 6(1)(f) GDPR. We have a legitimate interest in
        presenting our website as reliably and economically, in terms of data, as possible.
      </p>

      <h2 class="text-xl font-semibold">3. General notes and mandatory information</h2>

      <h3 class="text-lg font-semibold">Data protection</h3>
      <p>
        The operators of these pages take the protection of your personal data very seriously. We
        treat your personal data as confidential and in accordance with the statutory data
        protection regulations and this privacy policy.
      </p>
      <p>
        When you use this website, various personal data is collected. Personal data is any data
        that can be used to identify you personally. This privacy policy explains what data we
        collect and what we use it for. It also explains how and for what purpose this happens.
      </p>
      <p>
        We point out that data transmission over the internet (for example, when communicating by
        email) can have security gaps. Complete protection of data against access by third parties
        is not possible.
      </p>

      <h3 id="responsible" class="text-lg font-semibold">Information about the controller</h3>
      <p>The controller responsible for data processing on this website is:</p>
      <p>
        Simon Ostendorf<br />
        Frankenstr. 8<br />
        52070 Aachen, Germany
      </p>
      <p>
        Phone:
        <a class="text-accent-text hover:underline" href="tel:+4915730971784">+49 1573 0971784</a
        ><br />
        Email:
        <a class="text-accent-text hover:underline" href="mailto:simon@ostendorf.tech"
          >simon@ostendorf.tech</a
        >
      </p>
      <p>
        The controller is the natural or legal person who alone, or jointly with others, determines
        the purposes and means of the processing of personal data (such as names, email addresses,
        and the like).
      </p>

      <h3 class="text-lg font-semibold">Storage period</h3>
      <p>
        Unless a more specific storage period is stated within this privacy policy, your personal
        data will remain with us until the purpose for the data processing no longer applies. If you
        assert a legitimate request for deletion or withdraw your consent to data processing, your
        data will be deleted, unless we have other legally permissible grounds for storing your
        personal data (for example, retention periods under tax or commercial law); in the latter
        case, deletion will take place once those grounds cease to apply.
      </p>

      <h3 class="text-lg font-semibold">
        General notes on the legal grounds for data processing on this website
      </h3>
      <p>
        Where you have given consent to data processing, we process your personal data on the basis
        of Art. 6(1)(a) GDPR, or Art. 9(2)(a) GDPR where special categories of data under Art. 9(1)
        GDPR are processed. Where you have given explicit consent to the transfer of personal data
        to third countries, data processing is additionally carried out on the basis of Art.
        49(1)(a) GDPR. Where you have consented to the storage of cookies or to access to
        information on your device (for example, via device fingerprinting), data processing is
        additionally carried out on the basis of Section 25(1) TDDDG (the German
        Telecommunications-Digital-Services-Data-Protection Act). Consent can be withdrawn at any
        time. Where your data is required for the performance of a contract or to carry out
        pre-contractual measures, we process your data on the basis of Art. 6(1)(b) GDPR. We further
        process your data where this is necessary to comply with a legal obligation, on the basis of
        Art. 6(1)(c) GDPR. Data processing may also be carried out on the basis of our legitimate
        interest under Art. 6(1)(f) GDPR. The legal grounds relevant in each individual case are set
        out in the following paragraphs of this privacy policy.
      </p>

      <h3 class="text-lg font-semibold">Recipients of personal data</h3>
      <p>
        In the course of our business activities, we work together with various external parties.
        This sometimes requires the transfer of personal data to these external parties. We only
        pass on personal data to external parties where this is necessary for the performance of a
        contract, where we are legally obliged to do so (for example, passing data to tax
        authorities), where we have a legitimate interest under Art. 6(1)(f) GDPR in doing so, or
        where some other legal basis permits the disclosure. Where we use processors, we only pass
        on our customers' personal data on the basis of a valid data processing agreement. In the
        case of joint processing, an agreement on joint controllership is concluded.
      </p>

      <h3 class="text-lg font-semibold">Withdrawing your consent to data processing</h3>
      <p>
        Many data processing operations are only possible with your express consent. You can
        withdraw consent you have already given at any time. The lawfulness of data processing
        carried out before the withdrawal remains unaffected by the withdrawal.
      </p>

      <h3 class="text-lg font-semibold">
        Right to object to the collection of data in special cases, and to direct marketing (Art. 21
        GDPR)
      </h3>
      <p class="uppercase">
        Where data processing is based on Art. 6(1)(e) or (f) GDPR, you have the right to object at
        any time, on grounds relating to your particular situation, to the processing of your
        personal data; this also applies to profiling based on these provisions. The legal basis on
        which any processing is based can be found in this privacy policy. If you object, we will no
        longer process your personal data concerned, unless we can demonstrate compelling legitimate
        grounds for the processing which override your interests, rights and freedoms, or the
        processing serves the establishment, exercise or defence of legal claims (objection under
        Art. 21(1) GDPR).
      </p>
      <p class="uppercase">
        Where your personal data is processed for the purposes of direct marketing, you have the
        right to object at any time to the processing of personal data concerning you for the
        purposes of such marketing; this also applies to profiling to the extent it is related to
        such direct marketing. If you object, your personal data will subsequently no longer be used
        for the purposes of direct marketing (objection under Art. 21(2) GDPR).
      </p>

      <h3 class="text-lg font-semibold">
        Right to lodge a complaint with the competent supervisory authority
      </h3>
      <p>
        In the event of breaches of the GDPR, data subjects have the right to lodge a complaint with
        a supervisory authority, in particular in the member state of their habitual residence,
        place of work, or the place of the alleged breach. This right to complain exists without
        prejudice to any other administrative or judicial remedy.
      </p>

      <h3 class="text-lg font-semibold">Right to data portability</h3>
      <p>
        You have the right to have data that we process automatically on the basis of your consent
        or in performance of a contract handed over to you or to a third party in a commonly used,
        machine-readable format. Where you request the direct transfer of the data to another
        controller, this will only be done to the extent it is technically feasible.
      </p>

      <h3 class="text-lg font-semibold">Access, rectification and erasure</h3>
      <p>
        Within the scope of the applicable statutory provisions, you have the right at any time to
        free information about your stored personal data, its origin and recipients, and the purpose
        of the data processing, and, where applicable, a right to have this data corrected or
        deleted. You are welcome to contact us at any time regarding this and any further questions
        about personal data.
      </p>

      <h3 class="text-lg font-semibold">Right to restriction of processing</h3>
      <p>
        You have the right to request the restriction of the processing of your personal data. You
        are welcome to contact us at any time for this purpose. The right to restriction of
        processing exists in the following cases:
      </p>
      <ul class="list-disc pl-6">
        <li>
          If you dispute the accuracy of your personal data stored with us, we generally need time
          to verify this. For the duration of the review, you have the right to demand the
          restriction of the processing of your personal data.
        </li>
        <li>
          If the processing of your personal data was or is unlawful, you may demand the restriction
          of the data processing instead of its erasure.
        </li>
        <li>
          If we no longer need your personal data, but you need it to exercise, defend or assert
          legal claims, you have the right to demand the restriction of the processing of your
          personal data instead of its erasure.
        </li>
        <li>
          If you have lodged an objection under Art. 21(1) GDPR, a balancing of your interests and
          ours must take place. As long as it has not yet been determined whose interests prevail,
          you have the right to demand the restriction of the processing of your personal data.
        </li>
      </ul>
      <p>
        If you have restricted the processing of your personal data, this data may, other than being
        stored, only be processed with your consent, or for the establishment, exercise or defence
        of legal claims, or for the protection of the rights of another natural or legal person, or
        for reasons of important public interest of the European Union or a member state.
      </p>

      <h3 class="text-lg font-semibold">SSL or TLS encryption</h3>
      <p>
        For security reasons and to protect the transmission of confidential content, such as orders
        or enquiries you send to us as the site operator, this site uses SSL or TLS encryption. You
        can recognise an encrypted connection by the fact that the browser's address bar changes
        from "http://" to "https://" and by the lock symbol in your browser bar.
      </p>
      <p>
        When SSL or TLS encryption is activated, the data you transmit to us cannot be read by third
        parties.
      </p>

      <h3 class="text-lg font-semibold">Objection to promotional emails</h3>
      <p>
        We hereby object to the use of contact details published in the context of the obligation to
        provide an imprint for the purpose of sending unsolicited advertising and information
        material. The operators of these pages expressly reserve the right to take legal action in
        the event of the unsolicited sending of promotional information, for example through spam
        emails.
      </p>

      <h2 class="text-xl font-semibold">4. Data collection on this website</h2>

      <h3 class="text-lg font-semibold">Cookies</h3>
      <p>
        This website sets a single cookie: a technical session cookie, set on every page view where
        your browser does not already have one. It contains only a random identifier that cannot be
        traced back to you personally, and exists to let our server protect the contact form against
        cross-site request forgery (CSRF) attacks; it is not used for tracking, analytics or
        advertising, and is not transmitted to third parties.
      </p>
      <p>
        The cookie is a session cookie with no storage-duration attribute of its own: your browser
        discards it once you close it. It is also set so that no script on the page can read it
        (HttpOnly), so that it is only transmitted over an encrypted connection (Secure), and so
        that it is not sent along to another website (SameSite=Lax).
      </p>
      <p>
        In our assessment, this cookie is technically necessary within the meaning of Section 25(2)
        no. 2 TDDDG, because the contact form cannot be effectively protected against cross-site
        request forgery without a session cookie of this kind; consent is therefore not required,
        and we do not display a cookie banner. Beyond this, we do not store any further information
        on your device, do not access information stored there, and do not use device fingerprinting
        in the browser either.
      </p>

      <h3 class="text-lg font-semibold">Server log files</h3>
      <p>
        For every request to this website, our web server automatically writes one log line. It
        records:
      </p>
      <ul class="list-disc pl-6">
        <li>the time of the request</li>
        <li>the IP address</li>
        <li>
          a random request id our server assigns, so that log lines belonging to one request can be
          matched to each other
        </li>
        <li>the request method (for example, GET or POST)</li>
        <li>
          a coarse, internal category for the request (for example, "a page of this website", "an
          asset file"), not the full address requested with all its parameters; for an ordinary page
          view, this category does not even distinguish which page was viewed – the home page, the
          imprint, the privacy policy, the AGB and the error page all get the same value
        </li>
        <li>the HTTP status code and the amount of data transferred</li>
        <li>how long the request took to answer</li>
      </ul>
      <p>
        Your device's browser type (user agent) and any referrer URL are not recorded as part of
        this.
      </p>
      <p>This data is not merged with other data sources.</p>
      <p>
        This data is collected on the basis of Art. 6(1)(f) GDPR. The website operator has a
        legitimate interest in the technically error-free presentation, security and optimisation of
        its website, and in defending against misuse; the server log files must be collected for
        this purpose.
      </p>
      <p>
        The log files are retained for a limited period only and are automatically deleted after 30
        days at the latest. Log files whose further retention is required for evidentiary purposes
        are excluded from deletion until the incident in question has been finally clarified.
      </p>

      <h3 class="text-lg font-semibold">Rate limiting</h3>
      <p>
        To protect our website against overload and misuse, we limit how many requests a single IP
        address can make to us within a short period of time. To do this, our server counts, for
        each address, how many requests it has recently received; where an address makes too many
        requests, further requests from it are briefly refused.
      </p>
      <p>
        This counter exists only in our server's memory. It is not written to disk and not
        transmitted to third parties. Once an address has made no further requests for a while, our
        server automatically discards the associated counter; no further retention takes place.
      </p>
      <p>
        This general rate limiting applies to every request to our website and is independent of the
        limiting described in the "Contact form" section for the contact form specifically; the two
        are counted separately.
      </p>
      <p>
        This processing is carried out on the basis of Art. 6(1)(f) GDPR. We have a legitimate
        interest in protecting our website against overload and abusive access.
      </p>

      <h3 id="umami" class="text-lg font-semibold">Visitor statistics with Umami</h3>
      <p>
        To understand which content on our website is read and how visitors find us, we compile
        visitor statistics using the open-source software Umami. We operate Umami ourselves on our
        own servers in Germany (see "Hosting on our own servers"). We store backup copies of the
        statistics database with Hetzner Online GmbH, Industriestr. 25, 91710 Gunzenhausen, in a
        data centre in Germany; beyond this, the data is not transmitted to third parties.
      </p>
      <p>
        Umami does not run in your browser: no script is loaded, no cookie is set, and nothing is
        stored on or read from your device. Instead, our web server reports every page request to
        Umami, using the information your browser transmits to us in any case when making the
        request:
      </p>
      <ul class="list-disc pl-6">
        <li>the page requested (without any attached parameters) and its title</li>
        <li>
          the referrer URL without parameters, where you come from another website and your browser
          transmits it
        </li>
        <li>your browser's preferred language</li>
        <li>browser type and version, and operating system used (user agent)</li>
        <li>IP address</li>
      </ul>
      <p>
        Umami derives the country, region and city from the IP address. From the IP address, the
        user agent, a secret key and the current month, Umami forms a pseudonymous identifier to
        group together the page views of one visit; the identifier therefore changes every month.
        Umami does not store the IP address or the full user agent, only the information derived
        from them (country, region, city, browser, operating system) together with the pseudonymous
        identifier. No merging with other data, and no identification of individual visitors, takes
        place.
      </p>
      <p>
        The statistics data is automatically deleted after 14 months. In the backup copies of the
        database, it typically remains for up to a further 15 days afterwards.
      </p>
      <p>
        Requests where your browser sends the "Do Not Track" (DNT) or "Global Privacy Control" (GPC)
        signal are not recorded, nor are requests your browser merely preloads.
      </p>
      <p>
        This processing is carried out on the basis of Art. 6(1)(f) GDPR. We have a legitimate
        interest in statistically evaluating the use of our website in order to improve our
        offering, without passing on our visitors' data to third parties in doing so. You can object
        to this processing at any time by activating "Do Not Track" or "Global Privacy Control" in
        your browser, or by contacting us.
      </p>

      <h3 class="text-lg font-semibold">Contact form</h3>
      <p>
        If you send us enquiries using the contact form, your details from the enquiry form,
        including the contact details you provide there (name, email address and message), are
        stored with us for the purpose of processing the enquiry and in case of follow-up questions.
        We do not pass on this data without your consent.
      </p>
      <p>
        Your details are not stored by our web server in a database, but sent as an email via our
        own mail server to our mailbox. We operate the mail server ourselves on a server we rent
        from Hetzner Online GmbH, Industriestr. 25, 91710 Gunzenhausen, in a data centre in Germany.
        Transmission from the web server to the mail server is encrypted.
      </p>
      <p>
        To protect against spam, your browser solves a small computational task (a so-called proof
        of work) when you fill in the form. To do this, it requests a random task from our server
        and sends the solution back together with the form. This itself sets no further cookies,
        stores no further information on your device, and uses no third-party services for this
        purpose; the technical session cookie set when you load any page of this website (see the
        "Cookies" section) also protects this form against cross-site request forgery. To prevent
        misuse, our server also processes your IP address briefly in memory in order to limit the
        number of requests per address; it is not stored permanently for this purpose. The legal
        basis for this is our legitimate interest in protection against spam and misuse (Art.
        6(1)(f) GDPR).
      </p>
      <p>
        Processing of this data is carried out on the basis of Art. 6(1)(b) GDPR, where your enquiry
        relates to the performance of a contract or is necessary to carry out pre-contractual
        measures. In all other cases, processing is based on our legitimate interest in the
        effective handling of enquiries addressed to us (Art. 6(1)(f) GDPR), or on your consent
        (Art. 6(1)(a) GDPR) where this has been requested; consent can be withdrawn at any time.
      </p>
      <p>
        The data you enter in the contact form remains with us until you ask us to delete it,
        withdraw your consent to its storage, or the purpose for storing the data no longer applies
        (for example, after your enquiry has been fully processed). Mandatory statutory provisions,
        in particular retention periods, remain unaffected.
      </p>

      <h3 class="text-lg font-semibold">Enquiry by email, phone or fax</h3>
      <p>
        If you contact us by email, phone or fax, your enquiry, including all personal data arising
        from it (name, enquiry), is stored and processed by us for the purpose of handling your
        matter. We do not pass on this data without your consent.
      </p>
      <p>
        Processing of this data is carried out on the basis of Art. 6(1)(b) GDPR, where your enquiry
        relates to the performance of a contract or is necessary to carry out pre-contractual
        measures. In all other cases, processing is based on our legitimate interest in the
        effective handling of enquiries addressed to us (Art. 6(1)(f) GDPR), or on your consent
        (Art. 6(1)(a) GDPR) where this has been requested; consent can be withdrawn at any time.
      </p>
      <p>
        The data you send to us in enquiries remains with us until you ask us to delete it, withdraw
        your consent to its storage, or the purpose for storing the data no longer applies (for
        example, after your matter has been fully handled). Mandatory statutory provisions, in
        particular statutory retention periods, remain unaffected.
      </p>

      <h2 class="text-xl font-semibold">5. Social media</h2>

      <h3 class="text-lg font-semibold">Instagram posts on this website</h3>
      <p>
        In the "Impressions" section, we display the latest posts from our Instagram account
        @simonostendorf.vtech. Instagram is provided by Meta Platforms Ireland Limited, Merrion
        Road, Dublin 4, D04 X2K5, Ireland.
      </p>
      <p>
        Our server regularly retrieves the posts (images, image descriptions and links) via
        Instagram's application programming interface (API), holds them on our server and delivers
        them from our own domain. Your browser does not establish any connection to Instagram's or
        Meta's servers in doing so, and no data about your visit is transmitted to Instagram or
        Meta. In making this retrieval, our server only transmits its own IP address and our access
        key to Instagram.
      </p>
      <p>
        Only once you click on a post does it open on Instagram's website. From that point,
        Instagram's privacy policy applies:
        <a
          class="text-accent-text hover:underline"
          href="https://privacycenter.instagram.com/policy/"
          target="_blank"
          rel="noopener noreferrer"
          >https://privacycenter.instagram.com/policy/</a
        >.
      </p>
      <p>
        The display of the posts is carried out on the basis of Art. 6(1)(f) GDPR. We have a
        legitimate interest in illustrating our work, without passing on our visitors' data to third
        parties in doing so.
      </p>

      <h3 class="text-lg font-semibold">Links to Instagram, Facebook and WhatsApp</h3>
      <p>
        On this website, we link to our profiles on Instagram and Facebook, and to WhatsApp. The
        provider of Instagram and Facebook is Meta Platforms Ireland Limited, Merrion Road, Dublin
        4, D04 X2K5, Ireland; the provider of WhatsApp is WhatsApp Ireland Limited, Merrion Road,
        Dublin 4, D04 X2K5, Ireland.
      </p>
      <p>
        These are simple links. Plugins, "like" buttons or other elements of these services are not
        embedded on our website, so no data is transmitted to these providers when you visit our
        website. Only once you click on a link are you forwarded to the respective provider's page;
        from that point, that provider's privacy policy applies:
        <a
          class="text-accent-text hover:underline"
          href="https://privacycenter.instagram.com/policy/"
          target="_blank"
          rel="noopener noreferrer"
          >Instagram</a
        >,
        <a
          class="text-accent-text hover:underline"
          href="https://www.facebook.com/privacy/policy/"
          target="_blank"
          rel="noopener noreferrer"
          >Facebook</a
        >
        and
        <a
          class="text-accent-text hover:underline"
          href="https://www.whatsapp.com/legal/privacy-policy-eea"
          target="_blank"
          rel="noopener noreferrer"
          >WhatsApp</a
        >.
      </p>
      <p>
        If you write to us via WhatsApp, we process your phone number and your message as described
        in the section "Enquiry by email, phone or fax". The message is transmitted via WhatsApp's
        servers; WhatsApp's privacy policy applies to this.
      </p>
    </div>
  </GovuePage>
</template>
