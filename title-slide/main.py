from manim import *
import numpy as np

class HippocampusTitle(Scene):
    def construct(self):
        # Dark background
        self.camera.background_color = "#0a0e1a"
        
        # Main title - the star of the show
        title = Text("Hippocampus", font_size=84, weight=BOLD)
        title.set_color_by_gradient("#ff6b35", "#ffd23f")
        
        # Simple tagline
        tagline = Text("AI Agent Memory That Actually Works", font_size=36)
        tagline.set_color("#88c999")
        
        # Subtitle
        subtitle = Text("The SQLite Database for AI Agents", font_size=28, slant=ITALIC)
        subtitle.set_color("#b8b8b8")
        
        # Animated brain - simple connected dots
        brain = VGroup()
        
        # Create brain nodes
        nodes = VGroup()
        node_positions = [
            [-1.5, 0.5, 0], [-0.5, 1, 0], [0.5, 0.8, 0], [1.2, 0.2, 0],
            [-1.2, -0.3, 0], [-0.3, -0.1, 0], [0.8, -0.5, 0], [1.5, -0.8, 0],
            [0, 0.3, 0], [-0.8, 0.1, 0]
        ]
        
        for pos in node_positions:
            node = Dot(radius=0.08, color="#ff6b35")
            node.move_to(pos)
            nodes.add(node)
        
        # Create connections between nearby nodes
        connections = VGroup()
        for i, node1 in enumerate(nodes):
            for j, node2 in enumerate(nodes):
                if i < j and np.linalg.norm(np.array(node1.get_center()) - np.array(node2.get_center())) < 1.5:
                    line = Line(node1.get_center(), node2.get_center(), 
                              stroke_width=2, color="#88c999", stroke_opacity=0.6)
                    connections.add(line)
        
        brain.add(connections, nodes)
        brain.scale(0.6)
        
        # Position everything vertically
        brain.move_to(UP * 2.5)
        title.move_to(UP * 0.5)
        tagline.next_to(title, DOWN, buff=0.5)
        subtitle.next_to(tagline, DOWN, buff=0.3)
        
        # Animations
        self.play(
            Write(title, run_time=1.5),
            FadeIn(brain, run_time=2)
        )
        
        self.wait(0.5)
        
        # Tagline appears
        self.play(FadeIn(tagline, run_time=1))
        
        # Subtitle appears
        self.play(FadeIn(subtitle, run_time=1))
        
        # Brain nodes fire in sequence - like neural activity
        for i in range(3):  # Repeat the firing pattern 3 times
            # Random firing pattern
            firing_order = np.random.permutation(len(nodes))
            for j in firing_order:
                node = nodes[j]
                self.play(
                    node.animate.set_color("#ffd23f").scale(1.5),
                    run_time=0.15
                )
                self.play(
                    node.animate.set_color("#ff6b35").scale(1/1.5),
                    run_time=0.15
                )
        
        # Title pulse
        self.play(
            title.animate.scale(1.03),
            rate_func=there_and_back,
            run_time=1
        )
        
        self.wait(3)
