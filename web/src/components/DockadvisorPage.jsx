import {Footer} from "./Footer";
import {Container} from "./Container";
import {DockadvisorProvider} from "./DockadvisorContext";
import {DockadvisorScoreGauge, DockadvisorEditorWrapper} from "./DockadvisorClient";

export default function DockadvisorPage() {
  return (
    <>
      <main>
        <Container className="mt-12">
          <DockadvisorProvider>
            <div className="flex flex-col lg:flex-row items-end justify-between gap-8 mb-0 lg:mb-6">
              <div className="flex-1">
                <h1 className="max-w-4xl font-display text-3xl font-medium tracking-tight text-slate-900 sm:text-5xl">
                  Dockadvisor
                </h1>
                <h2 className="font-display text-lg tracking-tight text-slate-700 sm:text-xl">
                  Make your Dockerfile proud of itself.
                </h2>
                <p className="mt-4 text-base text-slate-600 max-w-3xl">
                  Dockadvisor is a free online Dockerfile analyzer built by <a href="https://deckrun.com" target="_blank" className="text-violet-600 hover:text-violet-500 underline">Deckrun</a> that helps you write better, more efficient Docker configurations.
                  Paste your Dockerfile to get instant feedback on best practices, issues, and optimization opportunities.
                </p>
              </div>
              <DockadvisorScoreGauge />
            </div>
            <DockadvisorEditorWrapper />

            <section className="mt-16 mb-8">
              <h2 className="text-2xl font-bold text-slate-900 mb-6">Why Dockerfile Optimization Matters</h2>
              <div className="prose prose-slate max-w-none">
                <p className="text-base text-slate-600 leading-relaxed mb-4">
                  This is not just about best practices; a well-optimized Dockerfile directly impacts your development workflow, security posture, and production costs. Here's why it matters:
                </p>
                <div className="grid md:grid-cols-2 gap-6 mt-6">
                  <div className="bg-white p-6 rounded-lg border border-slate-200">
                    <h3 className="text-lg font-semibold text-slate-900 mb-3">Faster Build Times</h3>
                    <p className="text-base text-slate-600 leading-relaxed">
                      Proper layer caching and instruction ordering can bring build times down from minutes to seconds.
                      Multi-stage builds remove all the extra dependencies from the final images, making them leaner and faster.
                    </p>
                  </div>
                  <div className="bg-white p-6 rounded-lg border border-slate-200">
                    <h3 className="text-lg font-semibold text-slate-900 mb-3">Smaller Image Sizes</h3>
                    <p className="text-base text-slate-600 leading-relaxed">
                      Optimizing Dockerfiles can lead to image sizes being reduced by 50-90%. Smaller images offer faster deployments,
                      lower storage costs, and reduced attack surface for security vulnerabilities.
                    </p>
                  </div>
                  <div className="bg-white p-6 rounded-lg border border-slate-200">
                    <h3 className="text-lg font-semibold text-slate-900 mb-3">Enhanced Security</h3>
                    <p className="text-base text-slate-600 leading-relaxed">
                      Security Exposed secret detection, running as non-root users, and avoiding deprecated features all help avert security
                      breaches. One exposed API key or password is enough to compromise an entire infrastructure.
                    </p>
                  </div>
                  <div className="bg-white p-6 rounded-lg border border-slate-200">
                    <h3 className="text-lg font-semibold text-slate-900 mb-3">Lower Costs</h3>
                    <p className="text-base text-slate-600 leading-relaxed">
                      Lower Costs Efficient Docker images reduce bandwidth costs, storage fees, and compute resources.
                      This savings compounds in production where hundreds or thousands of container instances may be used.
                    </p>
                  </div>
                </div>
              </div>
            </section>

            <section className="mt-16 mb-8">
              <h2 className="text-2xl font-bold text-slate-900 mb-6">Frequently Asked Questions</h2>
              <div className="space-y-6">
                <div>
                  <h3 className="text-lg font-semibold text-slate-900 mb-2">
                    What does Dockadvisor check in my Dockerfile?
                  </h3>
                  <p className="text-base text-slate-600 leading-relaxed">
                    Dockadvisor implements over 50 validation rules across 18 different Dockerfile instructions. It checks syntax errors
                    (JSON arrays, key=value formats, port ranges), security issues (exposed secrets and credentials), best practices
                    (deprecated features, proper signal handling), and style consistency (casing, absolute paths). It also performs
                    cross-instruction analysis to detect issues in multi-stage builds, variable scope problems, and duplicate declarations.
                  </p>
                </div>

                <div>
                  <h3 className="text-lg font-semibold text-slate-900 mb-2">
                    What security vulnerabilities can Dockadvisor detect?
                  </h3>
                  <p className="text-base text-slate-600 leading-relaxed">
                    Dockadvisor actively scans for exposed sensitive data in ARG and ENV instructions by detecting keywords like "password,"
                    "secret," "apikey," "token," and similar patterns. It validates RUN instruction mount types (bind, cache, tmpfs, secret, ssh)
                    and checks network/security flags. It also warns against security anti-patterns and helps prevent credential leakage in
                    your Docker images.
                  </p>
                </div>

                <div>
                  <h3 className="text-lg font-semibold text-slate-900 mb-2">
                    Does Dockadvisor work with multi-stage builds?
                  </h3>
                  <p className="text-base text-slate-600 leading-relaxed">
                    Yes! Dockadvisor has sophisticated multi-stage build analysis. It detects duplicate stage names, ensures ARG variables
                    in FROM instructions are properly declared in global scope with default values, tracks variable scope throughout each
                    build stage, and warns when multiple CMD, ENTRYPOINT, or HEALTHCHECK instructions appear (only the last one takes effect).
                  </p>
                </div>

                <div>
                  <h3 className="text-lg font-semibold text-slate-900 mb-2">
                    Is my Dockerfile sent to your servers?
                  </h3>
                  <p className="text-base text-slate-600 leading-relaxed">
                    No. Dockadvisor runs entirely in your browser using WebAssembly. Your Dockerfile never leaves your device,
                    ensuring complete privacy and security. All analysis happens client-side, making it safe to analyze proprietary
                    and sensitive Dockerfiles.
                  </p>
                </div>

                <div>
                  <h3 className="text-lg font-semibold text-slate-900 mb-2">
                    Is Dockadvisor free to use?
                  </h3>
                  <p className="text-base text-slate-600 leading-relaxed">
                    Yes, Dockadvisor is completely free for both personal and commercial use. No signup, no payment, no limits on usage.
                    We built this tool to help the Docker community write better, more secure Dockerfiles.
                  </p>
                </div>

                <div>
                  <h3 className="text-lg font-semibold text-slate-900 mb-2">
                    Which Dockerfile instructions does Dockadvisor analyze?
                  </h3>
                  <p className="text-base text-slate-600 leading-relaxed">
                    Dockadvisor analyzes all 18 major Dockerfile instructions including{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#from" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">FROM</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#run" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">RUN</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#cmd" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">CMD</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#entrypoint" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">ENTRYPOINT</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#copy" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">COPY</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#add" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">ADD</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#env" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">ENV</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#arg" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">ARG</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#workdir" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">WORKDIR</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#expose" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">EXPOSE</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#user" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">USER</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#label" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">LABEL</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#healthcheck" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">HEALTHCHECK</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#shell" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">SHELL</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#stopsignal" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">STOPSIGNAL</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#onbuild" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">ONBUILD</a>,{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#volume" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">VOLUME</a>, and the deprecated{' '}
                    <a href="https://docs.docker.com/reference/dockerfile/#maintainer-deprecated" target="_blank" rel="noopener noreferrer" className="text-violet-600 hover:text-violet-500 underline">MAINTAINER</a>.
                    It validates syntax, flags, and best practices for each instruction type.
                  </p>
                </div>
              </div>
            </section>

            <p className="m-12 text-center text-xs text-slate-400">
              Dockadvisor uses open source software including{' '}
              <a href="https://github.com/moby/buildkit" target="_blank" rel="noopener noreferrer" className="underline hover:text-slate-500">
                moby/buildkit
              </a>{' '}
              licensed under the{' '}
              <a href="https://www.apache.org/licenses/LICENSE-2.0" target="_blank" rel="noopener noreferrer" className="underline hover:text-slate-500">
                Apache License 2.0
              </a>.
            </p>
          </DockadvisorProvider>
        </Container>
      </main>
      <Footer />
    </>
  )
}
