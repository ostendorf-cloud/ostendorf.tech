<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { GovuePage } from '@ostendorf-cloud/govue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const germanHref = computed(() => router.resolve({ name: 'agb', params: { locale: 'de' } }).href)
</script>

<template>
  <GovuePage width="reading" :title="t('legal.agb.title')">
    <div v-if="route.params.locale === 'de'" class="flex flex-col gap-4 text-text">
      <h2 class="text-xl font-semibold">§ 1 Geltungsbereich</h2>
      <p>
        Diese Allgemeinen Geschäftsbedingungen gelten für alle Verträge zwischen Simon Ostendorf
        (nachfolgend „Auftragnehmer“) und dem jeweiligen Auftraggeber (nachfolgend „Veranstalter“)
        über die Erbringung von tontechnischen Dienstleistungen sowie die Bereitstellung von
        Equipment. Abweichende Bedingungen des Veranstalters werden nur anerkannt, wenn der
        Auftragnehmer diesen ausdrücklich schriftlich zugestimmt hat.
      </p>

      <h2 class="text-xl font-semibold">§ 2 Leistungsumfang</h2>
      <p>Der Leistungsumfang ergibt sich aus dem jeweiligen schriftlichen Angebot.</p>
      <p>
        Ausdrücklich nicht Bestandteil der Leistung sind Tätigkeiten als Sicherheitsbeauftragter für
        Veranstaltungstechnik (SiMA), Rigging, Traversenbau, Bühnenbau, Elektroinstallation sowie
        alle weiteren gewerke- oder erlaubnispflichtigen Tätigkeiten nach DGUV und MVStättVO. Der
        Auftragnehmer ist keine zugelassene Fachkraft für diese Bereiche. Die Verantwortung und
        Haftung für diese Bereiche liegt ausschließlich beim Veranstalter.
      </p>

      <h2 class="text-xl font-semibold">§ 3 Angebot und Vertragsschluss</h2>
      <p>
        Angebote des Auftragnehmers sind freibleibend und für 14 Tage ab Ausstellungsdatum gültig,
        sofern keine andere Frist angegeben ist. Der Vertrag kommt durch schriftliche
        Auftragsbestätigung (E-Mail genügt) oder durch Beginn der Leistungserbringung zustande.
        Mündliche Nebenabreden bedürfen der schriftlichen Bestätigung.
      </p>

      <h2 class="text-xl font-semibold">§ 4 Vergütung und Zahlungsbedingungen</h2>
      <p>
        Die vereinbarte Vergütung ist je Veranstaltungstag fällig. Rechnungen sind innerhalb von 14
        Tagen nach Rechnungsstellung ohne Abzug zahlbar. Bei Konzertreihen oder Touren mit mehreren
        Terminen behält sich der Auftragnehmer eine Abschlagszahlung von 30 % vor Tourstart vor.
      </p>
      <p>Gemäß § 19 UStG wird aufgrund der Kleinunternehmerregelung keine Umsatzsteuer erhoben.</p>
      <p>
        Bei Zahlungsverzug ist der Auftragnehmer berechtigt, Verzugszinsen in Höhe von 5 % p.a. über
        dem jeweils geltenden Basiszinssatz zu berechnen.
      </p>

      <h2 class="text-xl font-semibold">§ 5 Stornierung und Ausfallhonorar</h2>
      <p>
        Bei Absage oder Verschiebung eines oder mehrerer Termine durch den Veranstalter gelten
        folgende Stornierungsgebühren bezogen auf das vereinbarte Honorar je abgesagtem Termin:
      </p>
      <ul class="list-disc pl-6">
        <li>Absage mehr als 60 Tage vor dem Termin: 25 % des Honorars</li>
        <li>Absage 30–60 Tage vor dem Termin: 50 % des Honorars</li>
        <li>Absage 14–29 Tage vor dem Termin: 75 % des Honorars</li>
        <li>Absage weniger als 14 Tage vor dem Termin: 100 % des Honorars</li>
      </ul>
      <p>
        Bereits entstandene Kosten (z. B. Equipment-Anschaffungen oder -Mieten, die nachweislich für
        die gebuchte Produktion getätigt wurden) sind unabhängig vom Zeitpunkt der Absage
        vollständig zu erstatten. Der Auftragnehmer ist verpflichtet, anderweitig erzielte Einnahmen
        für den abgesagten Zeitraum anzurechnen.
      </p>

      <h2 class="text-xl font-semibold">§ 6 Arbeitsbedingungen und Arbeitszeit</h2>
      <p>
        Ein Veranstaltungstag (Tagessatz) umfasst bis zu 11 Stunden Arbeitszeit inklusive einer
        Stunde Pause. Wird diese Arbeitszeit überschritten, wird ein zusätzlicher Tagessatz in
        gleicher Höhe berechnet, sofern keine abweichende Vereinbarung getroffen wurde.
      </p>
      <p>
        Der Auftragnehmer ist berechtigt, seine Tätigkeit zu unterbrechen oder einzustellen, wenn
        die Arbeitsbedingungen vor Ort seine Sicherheit oder die Sicherheit des Equipments gefährden
        und der Veranstalter trotz Aufforderung keine Abhilfe schafft. In diesem Fall besteht kein
        Anspruch des Veranstalters auf Rückerstattung bereits geleisteter Zahlungen.
      </p>

      <h2 class="text-xl font-semibold">§ 7 Equipment – Bereitstellung und Haftung</h2>
      <p>
        Das vom Auftragnehmer mitgebrachte Equipment wird in einwandfreiem, funktionstüchtigem
        Zustand zur Verfügung gestellt. Der Veranstalter haftet für Beschädigungen oder Verlust des
        Equipments ab dem Zeitpunkt der Anlieferung bis zum Abtransport. Dies gilt insbesondere für
        Schäden durch unsachgemäße Handhabung durch Dritte (Publikum, lokales Personal), Diebstahl
        sowie Elementarschäden.
      </p>
      <p>
        Beschädigte oder verlorene Geräte werden zum aktuellen Wiederbeschaffungspreis (Neupreis) in
        Rechnung gestellt. Eventuelle Schäden sind unverzüglich, spätestens bei Abbau zu melden. Der
        Auftragnehmer empfiehlt dem Veranstalter den Abschluss einer
        Veranstalterhaftpflichtversicherung.
      </p>
      <p>
        Der Auftragnehmer haftet nicht für Schäden, die durch mangelhafte Stromversorgung
        (Schwankungen, Überspannung) seitens des Veranstalters oder der Venue entstehen.
      </p>

      <h2 class="text-xl font-semibold">§ 8 Pflichten des Veranstalters</h2>
      <p>Der Veranstalter stellt folgende Voraussetzungen kostenfrei zur Verfügung:</p>
      <ul class="list-disc pl-6">
        <li>Freier und rechtzeitiger Zugang zur Venue ab dem vereinbarten Get-in-Zeitpunkt</li>
        <li>
          Ausreichend dimensionierter Bühnenstrom (230V/16A Schuko, getrennt von Lichtstrom) sowie
          FOH-Strom
        </li>
        <li>FOH-Position im Zuschauerraum (nicht in Regiekabine), mind. 1 m Breite</li>
        <li>Netzwerkverbindung (Cat 5e oder höher) zwischen FOH und Bühne, sofern vereinbart</li>
        <li>
          Lokales technisches Personal (Licht- und Haustontechniker, Stagehands) gemäß
          Bühnenanweisung
        </li>
        <li>
          Einholung sämtlicher behördlicher Genehmigungen, insbesondere Schallschutzgenehmigungen
          und Ausnahmegenehmigungen für Lärmpegel
        </li>
        <li>Beauftragung und Kosten für etwaige Schallpegelmessungen oder Lärmschutzgutachten</li>
        <li>Einhaltung und Überwachung der behördlich zugelassenen Emissionsgrenzwerte</li>
      </ul>
      <p>
        Der Auftragnehmer wird auf Anweisung des Veranstalters auf Pegelgrenzen Rücksicht nehmen,
        trägt jedoch keine Verantwortung und keine Haftung für behördliche Maßnahmen, die aus der
        Überschreitung genehmigter Grenzwerte resultieren.
      </p>
      <p>
        Eventuelle GEMA-Gebühren und behördliche Genehmigungen trägt der Veranstalter. Bei
        Konzertreihen sind die Venue-Specs spätestens 4 Wochen vor der jeweiligen Veranstaltung an
        den Auftragnehmer zu übermitteln.
      </p>

      <h2 class="text-xl font-semibold">§ 9 Höhere Gewalt</h2>
      <p>
        Kann der Auftragnehmer seine Leistung aufgrund höherer Gewalt (Naturkatastrophen,
        behördliche Anordnungen, Stromausfall, Pandemie o. ä.) nicht erbringen, ist er von der
        Leistungspflicht befreit. Bereits geleistete Zahlungen für den betroffenen Zeitraum werden
        anteilig erstattet. Weitergehende Schadensersatzansprüche sind in diesem Fall
        ausgeschlossen.
      </p>

      <h2 class="text-xl font-semibold">§ 10 Referenzen und Bildrechte</h2>
      <p>
        Der Auftragnehmer ist berechtigt, die erbrachte Leistung zu Referenz- und Werbezwecken zu
        erwähnen. Sofern der Veranstalter nicht schriftlich widerspricht, darf der Auftragnehmer
        eigene Aufnahmen vom Arbeitsumfeld (nicht vom Künstler oder der Show selbst) zu
        Portfoliozwecken verwenden.
      </p>

      <h2 class="text-xl font-semibold">§ 11 Datenschutz</h2>
      <p>
        Im Zusammenhang mit der Anbahnung und Durchführung von Verträgen erhebt und speichert der
        Auftragnehmer personenbezogene Daten des Veranstalters ausschließlich zum Zweck der
        Vertragserfüllung. Eine Weitergabe an Dritte erfolgt nicht, es sei denn, dies ist gesetzlich
        vorgeschrieben. Auf Wunsch werden die Daten nach Abschluss der Geschäftsbeziehung gelöscht,
        soweit keine gesetzlichen Aufbewahrungsfristen entgegenstehen.
      </p>

      <h2 class="text-xl font-semibold">§ 12 Schlussbestimmungen</h2>
      <p>
        Es gilt das Recht der Bundesrepublik Deutschland. Gerichtsstand für alle Streitigkeiten ist
        Aachen, soweit gesetzlich zulässig. Sollten einzelne Bestimmungen dieser AGB unwirksam sein
        oder werden, bleibt die Wirksamkeit der übrigen Bestimmungen unberührt. An Stelle der
        unwirksamen Bestimmung gilt die gesetzlich zulässige Regelung, die dem ursprünglichen Zweck
        am nächsten kommt.
      </p>
      <p>
        Mündliche Nebenabreden bestehen nicht. Änderungen und Ergänzungen bedürfen der Schriftform.
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
