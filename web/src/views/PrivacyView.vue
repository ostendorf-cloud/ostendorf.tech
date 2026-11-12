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
        Kurz vorab: Diese Website verwendet keine Cookies, keine Tracking- oder Werbedienste und
        bindet keine Inhalte von Drittanbietern ein. Ihr Browser lädt alle Inhalte ausschließlich
        von unserem eigenen Server. Eine Besucherstatistik erstellen wir selbst auf unseren eigenen
        Servern, ohne Skript in Ihrem Browser und ohne etwas auf Ihrem Endgerät zu speichern.
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

      <h3 class="text-lg font-semibold">Keine Cookies</h3>
      <p>
        Diese Website verwendet keine Cookies. Wir speichern keine Informationen in Ihrem Endgerät
        und greifen nicht auf dort gespeicherte Informationen zu; auch Device-Fingerprinting im
        Browser setzen wir nicht ein. Eine Einwilligung nach § 25 Abs. 1 TDDDG ist deshalb nicht
        erforderlich, und wir zeigen kein Cookie-Banner an.
      </p>

      <h3 class="text-lg font-semibold">Server-Log-Dateien</h3>
      <p>
        Beim Aufruf dieser Website erheben und speichern unser Webserver und der vorgeschaltete
        Reverse-Proxy automatisch Informationen in so genannten Server-Log-Dateien, die Ihr Browser
        automatisch an uns übermittelt. Dies sind:
      </p>
      <ul class="list-disc pl-6">
        <li>IP-Adresse</li>
        <li>Datum und Uhrzeit der Serveranfrage</li>
        <li>Anfragezeile (aufgerufene Seite bzw. Datei, Anfragemethode und Protokollversion)</li>
        <li>HTTP-Statuscode und übertragene Datenmenge</li>
        <li>Browsertyp und Browserversion sowie verwendetes Betriebssystem (User-Agent)</li>
        <li>Referrer URL, sofern Ihr Browser sie übermittelt</li>
      </ul>
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
        sendet die Lösung zusammen mit dem Formular zurück. Dabei werden keine Cookies gesetzt,
        keine Informationen in Ihrem Endgerät gespeichert und keine Dienste von Drittanbietern
        eingebunden. Um Missbrauch zu verhindern, verarbeitet unser Server außerdem Ihre IP-Adresse
        kurzzeitig im Arbeitsspeicher, um die Zahl der Anfragen je Adresse zu begrenzen; dafür wird
        sie nicht dauerhaft gespeichert. Rechtsgrundlage hierfür ist unser berechtigtes Interesse am
        Schutz vor Spam und Missbrauch (Art. 6 Abs. 1 lit. f DSGVO).
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
    </div>
  </GovuePage>
</template>
